package network

import (
	"fmt"
	"io"
	goLog "log"
	"net"
	"net/http"
	"strings"
	"time"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/errors"
	"boscoin.io/sebak/lib/node"
	"github.com/gorilla/mux"
	logging "github.com/inconshreveable/log15"
	"golang.org/x/net/http2"
)

type Handlers map[string]func(http.ResponseWriter, *http.Request)

const (
	RouterNameNode   = "node"
	RouterNameAPI    = "api"
	RouterNameMetric = "metric"
	RouterNameDebug  = "debug"
)

var (
	UrlPathPrefixNode   = fmt.Sprintf("/%s", RouterNameNode)
	UrlPathPrefixAPI    = fmt.Sprintf("/%s", RouterNameAPI)
	UrlPathPrefixDebug  = fmt.Sprintf("/%s", RouterNameDebug)
	UrlPathPrefixMetric = fmt.Sprintf("/%s", RouterNameMetric)
)

type HTTP2MessageBroker struct {
	network *HTTP2Network
}

func (r HTTP2MessageBroker) Response(w io.Writer, o []byte) error {
	_, err := w.Write(o)
	return err
}

func (r HTTP2MessageBroker) Receive(msg common.NetworkMessage) {
	r.network.ReceiveChannel() <- msg
}

type HTTP2Network struct {
	tlsCertFile string
	tlsKeyFile  string

	server    *http.Server
	router    *mux.Router
	rootRoute *mux.Route

	receiveChannel chan common.NetworkMessage

	messageBroker MessageBroker
	ready         bool

	watchers []func(Network, net.Conn, http.ConnState)
	routers  map[string]*mux.Router
	handlers map[string]func(http.ResponseWriter, *http.Request)

	config *HTTP2NetworkConfig
	node   *node.LocalNode
	log    logging.Logger
}

type HandlerFunc func(w http.ResponseWriter, r *http.Request)

func NewHTTP2Network(config *HTTP2NetworkConfig) (h2n *HTTP2Network) {
	httpLog := log.New(logging.Ctx{"module": "http", "node": config.NodeName})
	errorLog := goLog.New(HTTP2ErrorLog15Writer{httpLog}, "", 0)

	server := &http.Server{
		Addr:              config.Addr,
		ReadTimeout:       config.ReadTimeout,
		ReadHeaderTimeout: config.ReadHeaderTimeout,
		WriteTimeout:      config.WriteTimeout,
		ErrorLog:          errorLog,
	}
	server.SetKeepAlivesEnabled(true)

	http2.ConfigureServer(
		server,
		&http2.Server{
			// MaxConcurrentStreams
			// MaxReadFrameSize
			// IdleTimeout
			IdleTimeout: config.IdleTimeout,
		},
	)

	baseRouter := mux.NewRouter()

	h2n = &HTTP2Network{
		server:         server,
		router:         baseRouter,
		tlsCertFile:    config.TLSCertFile,
		tlsKeyFile:     config.TLSKeyFile,
		receiveChannel: make(chan common.NetworkMessage),
		log:            httpLog,
	}
	h2n.handlers = map[string]func(http.ResponseWriter, *http.Request){}
	h2n.routers = map[string]*mux.Router{
		RouterNameNode:   baseRouter.PathPrefix(UrlPathPrefixNode).Subrouter(),
		RouterNameAPI:    baseRouter.PathPrefix(UrlPathPrefixAPI).Subrouter(),
		RouterNameMetric: baseRouter.PathPrefix(UrlPathPrefixMetric).Subrouter(),
		RouterNameDebug:  baseRouter.PathPrefix(UrlPathPrefixDebug).Subrouter(),
	}

	h2n.config = config

	h2n.setNotReadyHandler()
	h2n.server.ConnState = h2n.ConnState

	h2n.SetMessageBroker(HTTP2MessageBroker{network: h2n})

	return
}

// GetClient creates new keep-alive HTTP2 client
func (t *HTTP2Network) GetClient(endpoint *common.Endpoint) NetworkClient {
	rawClient, _ := common.NewHTTP2Client(defaultTimeout, 0, true)

	client := NewHTTP2NetworkClient(endpoint, rawClient)

	headers := http.Header{}
	headers.Set("User-Agent", fmt.Sprintf("v-%s", t.config.NodeName))
	client.SetDefaultHeaders(headers)

	return client
}

func (t *HTTP2Network) Endpoint() *common.Endpoint {
	return t.config.Endpoint
}

func (t *HTTP2Network) AddWatcher(f func(Network, net.Conn, http.ConnState)) {
	t.watchers = append(t.watchers, f)
}

func (t *HTTP2Network) ConnState(c net.Conn, state http.ConnState) {
	for _, f := range t.watchers {
		go f(t, c, state)
	}
}

func (t *HTTP2Network) setNotReadyHandler() {
	t.rootRoute = t.router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if !t.ready {
			http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
			return
		}
	})

	t.server.Handler = HTTP2Log15Handler{log: t.log, handler: t.router}
}

func (t *HTTP2Network) AddMiddleware(routerName string, mws ...mux.MiddlewareFunc) error {
	var r *mux.Router
	if len(routerName) < 1 {
		r = t.router
	} else {
		var ok bool
		if r, ok = t.routers[routerName]; !ok {
			return errors.ErrorNotMatcHTTPRouter
		}
	}
	for _, mw := range mws {
		r.Use(mw)
	}
	return nil
}

func (t *HTTP2Network) AddHandler(pattern string, handler http.HandlerFunc) (router *mux.Route) {
	var routerName string
	var prefix string
	switch {
	case strings.HasPrefix(pattern, UrlPathPrefixNode):
		routerName = RouterNameNode
		prefix = pattern[len(UrlPathPrefixNode):]
	case strings.HasPrefix(pattern, UrlPathPrefixAPI):
		routerName = RouterNameAPI
		prefix = pattern[len(UrlPathPrefixAPI):]
	case strings.HasPrefix(pattern, UrlPathPrefixMetric):
		routerName = RouterNameMetric
		prefix = pattern[len(UrlPathPrefixMetric):]
	case strings.HasPrefix(pattern, UrlPathPrefixDebug):
		routerName = RouterNameDebug
		prefix = pattern[len(UrlPathPrefixDebug):]
	default:
		if pattern == "" || pattern == "/" {
			return t.rootRoute.Handler(handler)
		} else {
			return t.router.HandleFunc(pattern, handler)
		}
	}

	r, _ := t.routers[routerName]

	// if a pattern has a suffix *,the router sets path prefix and handler
	if strings.HasSuffix(prefix, "*") {
		pathPrefix := strings.TrimSuffix(prefix, "*")
		return r.PathPrefix(pathPrefix).Handler(handler)
	}
	return r.HandleFunc(prefix, handler)
}

func (t *HTTP2Network) SetMessageBroker(mb MessageBroker) {
	t.messageBroker = mb
}

func (t *HTTP2Network) MessageBroker() MessageBroker {
	return t.messageBroker
}

func (t *HTTP2Network) Ready() error {
	t.server.Handler = HTTP2Log15Handler{log: t.log, handler: t.router}

	t.ready = true

	return nil
}

func (t *HTTP2Network) IsReady() bool {
	client, err := common.NewHTTP2Client(50*time.Millisecond, 50*time.Millisecond, false)
	if err != nil {
		return false
	}

	h2n := NewHTTP2NetworkClient(t.Endpoint(), client)
	if _, err := h2n.GetNodeInfo(); err != nil {
		return false
	}

	return true
}

// Start will start `HTTP2Network`.
func (t *HTTP2Network) Start() (err error) {
	if strings.ToLower(t.config.Endpoint.Scheme) == "http" {
		return t.server.ListenAndServe()
	}

	err = t.server.ListenAndServeTLS(t.tlsCertFile, t.tlsKeyFile)
	if err == http.ErrServerClosed {
		err = nil
		return
	}

	return
}

func (t *HTTP2Network) Stop() {
	t.server.Close()
}

func (t *HTTP2Network) ReceiveChannel() chan common.NetworkMessage {
	return t.receiveChannel
}

func (t *HTTP2Network) ReceiveMessage() <-chan common.NetworkMessage {
	return t.receiveChannel
}
