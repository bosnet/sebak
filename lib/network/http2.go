package network

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	logging "github.com/inconshreveable/log15"
	"golang.org/x/net/http2"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/node"
)

type Handlers map[string]func(http.ResponseWriter, *http.Request)

const (
	RouterNameNode = "node"
	RouterNameAPI  = "api"
)

var (
	UrlPathPrefixNode = fmt.Sprintf("/%s", RouterNameNode)
	UrlPathPrefixAPI  = fmt.Sprintf("/%s", RouterNameAPI)
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

	server *http.Server
	router *mux.Router

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
	server := &http.Server{
		Addr:              config.Addr,
		ReadTimeout:       config.ReadTimeout,
		ReadHeaderTimeout: config.ReadHeaderTimeout,
		WriteTimeout:      config.WriteTimeout,
		ErrorLog:          config.ErrorLog,
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
		log:            log.New(logging.Ctx{"node": config.Node.Alias()}),
	}
	h2n.handlers = map[string]func(http.ResponseWriter, *http.Request){}
	h2n.routers = map[string]*mux.Router{
		RouterNameNode: baseRouter.PathPrefix(UrlPathPrefixNode).Subrouter(),
		RouterNameAPI:  baseRouter.PathPrefix(UrlPathPrefixAPI).Subrouter(),
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
	headers.Set("User-Agent", fmt.Sprintf("v-%s", t.config.Node.Alias()))
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
	t.router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if !t.ready {
			http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
			return
		}
	})

	t.server.Handler = Log15LoggingHandler{log: t.log, handler: t.router}
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
	default:
		// if unknown pattern, it will be attached to base router
		return t.router.HandleFunc(pattern, handler)
	}

	r, _ := t.routers[routerName]

	return r.HandleFunc(prefix, handler)
}

func (t *HTTP2Network) SetMessageBroker(mb MessageBroker) {
	t.messageBroker = mb
}

func (t *HTTP2Network) MessageBroker() MessageBroker {
	return t.messageBroker
}

func (t *HTTP2Network) Ready() error {
	t.server.Handler = Log15LoggingHandler{log: t.log, handler: t.router}

	t.ready = true

	return nil
}

func (t *HTTP2Network) IsReady() bool {
	client, err := common.NewHTTP2Client(50*time.Millisecond, 50*time.Millisecond, false)
	if err != nil {
		return false
	}
	defer client.Close()

	h2n := NewHTTP2NetworkClient(t.Endpoint(), client)
	if _, err := h2n.GetNodeInfo(); err != nil {
		return false
	}

	return true
}

// Start will start `HTTP2Network`.
func (t *HTTP2Network) Start() (err error) {
	defer func() {
		close(t.receiveChannel)
	}()

	if strings.ToLower(t.config.Endpoint.Scheme) == "http" {
		return t.server.ListenAndServe()
	}

	return t.server.ListenAndServeTLS(t.tlsCertFile, t.tlsKeyFile)
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

type Log15LoggingHandler struct {
	log     logging.Logger
	handler http.Handler
}

func (l Log15LoggingHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	uid := common.GenerateUUID()

	username := "-"
	if r.URL.User != nil {
		if name := r.URL.User.Username(); name != "" {
			username = name
		}
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}

	uri := r.RequestURI

	// Requests using the CONNECT method over HTTP/2.0 must use
	// the authority field (aka r.Host) to identify the target.
	// Refer: https://httpwg.github.io/specs/rfc7540.html#CONNECT
	if r.ProtoMajor == 2 && r.Method == "CONNECT" {
		uri = r.Host
	}
	if uri == "" {
		uri = r.URL.RequestURI()
	}

	l.log.Debug(
		"http-request",
		"request-id", uid,
		"referer", r.Referer(),
		"user-agent", r.UserAgent(),
		"method", r.Method,
		"host", host,
		"username", username,
		"uri", uri,
		"proto", r.Proto,
		"remote", r.RemoteAddr,
	)

	writer := &Log15ResponseWriter{w: w}
	l.handler.ServeHTTP(writer, r)

	l.log.Debug(
		"http-response",
		"request-id", uid,
		"status", writer.Status(),
		"size", writer.Size(),
	)
}

type Log15ResponseWriter struct {
	w      http.ResponseWriter
	status int
	size   int
}

func (l *Log15ResponseWriter) Header() http.Header {
	return l.w.Header()
}

func (l *Log15ResponseWriter) Write(b []byte) (int, error) {
	size, err := l.w.Write(b)
	l.size += size
	return size, err
}

func (l *Log15ResponseWriter) WriteHeader(s int) {
	l.w.WriteHeader(s)
	l.status = s
}

func (l *Log15ResponseWriter) Status() int {
	return l.status
}

func (l *Log15ResponseWriter) Size() int {
	return l.size
}

func (l *Log15ResponseWriter) Flush() {
	f, ok := l.w.(http.Flusher)
	if ok {
		f.Flush()
	}
}
