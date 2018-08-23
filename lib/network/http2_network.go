package sebaknetwork

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"boscoin.io/sebak/lib/common"
	"github.com/gorilla/handlers"

	"github.com/gorilla/mux"
	"golang.org/x/net/http2"
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

type HTTP2Network struct {
	ctx         context.Context
	tlsCertFile string
	tlsKeyFile  string

	server *http.Server
	router *mux.Router

	receiveChannel chan Message

	messageBroker MessageBroker
	ready         bool

	watchers []func(Network, net.Conn, http.ConnState)
	routers  map[string]*mux.Router
	handlers map[string]func(http.ResponseWriter, *http.Request)

	config HTTP2NetworkConfig
}

type MessageBroker interface {
	ResponseMessage(http.ResponseWriter, string)
	ReceiveMessage(*HTTP2Network, Message)
}

type HandlerFunc func(w http.ResponseWriter, r *http.Request)

func NewHTTP2Network(config HTTP2NetworkConfig) (h2n *HTTP2Network) {
	server := &http.Server{
		Addr:              config.Addr,
		ReadTimeout:       config.ReadTimeout,
		ReadHeaderTimeout: config.ReadHeaderTimeout,
		WriteTimeout:      config.WriteTimeout,
		// TODO replace custom logger
		//ErrorLog:
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
		receiveChannel: make(chan Message),
	}
	h2n.handlers = map[string]func(http.ResponseWriter, *http.Request){}
	h2n.routers = map[string]*mux.Router{
		RouterNameNode: baseRouter.PathPrefix(UrlPathPrefixNode).Subrouter(),
		RouterNameAPI:  baseRouter.PathPrefix(UrlPathPrefixAPI).Subrouter(),
	}

	h2n.config = config

	h2n.setNotReadyHandler()
	h2n.server.ConnState = h2n.ConnState

	h2n.SetMessageBroker(Http2MessageBroker{})

	return
}

type Http2MessageBroker struct{}

func (r Http2MessageBroker) ResponseMessage(w http.ResponseWriter, o string) {
	fmt.Fprintf(w, o)
}

func (r Http2MessageBroker) ReceiveMessage(t *HTTP2Network, msg Message) {
	t.ReceiveChannel() <- msg
}

func (t *HTTP2Network) Context() context.Context {
	return t.ctx
}

func (t *HTTP2Network) SetContext(ctx context.Context) {
	t.ctx = ctx
}

// GetClient creates new keep-alive HTTP2 client
func (t *HTTP2Network) GetClient(endpoint *sebakcommon.Endpoint) NetworkClient {
	rawClient, _ := sebakcommon.NewHTTP2Client(defaultTimeout, 0, true)

	client := NewHTTP2NetworkClient(endpoint, rawClient)

	headers := http.Header{}
	headers.Set("User-Agent", fmt.Sprintf("v-%s", t.config.NodeName))
	client.SetDefaultHeaders(headers)

	return client
}

func (t *HTTP2Network) Endpoint() *sebakcommon.Endpoint {
	host, port, _ := net.SplitHostPort(t.server.Addr)
	return &sebakcommon.Endpoint{Scheme: "https", Host: fmt.Sprintf("%s:%s", host, port)}
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

	t.server.Handler = handlers.CombinedLoggingHandler(t.config.HTTP2LogOutput, t.router)
}

func (t *HTTP2Network) AddHandler(ctx context.Context, args ...interface{}) (err error) {
	addAPIFunc := args[0].(func(context.Context, *HTTP2Network))
	addAPIFunc(ctx, t)
	return
}
func (t *HTTP2Network) AddAPIHandler(pattern string, handlerFunc http.HandlerFunc) (router *mux.Route) {
	apiRouter := t.routers[RouterNameAPI]
	return apiRouter.HandleFunc(pattern, handlerFunc)
}

func (t *HTTP2Network) SetMessageBroker(mb MessageBroker) {
	t.messageBroker = mb
}

func (t *HTTP2Network) Ready() error {
	nodeRouter := t.routers[RouterNameNode]
	nodeRouter.HandleFunc("/", NodeInfoHandler(t.Context(), t))
	nodeRouter.HandleFunc("/connect", ConnectHandler(t.Context(), t)).Methods("POST")
	nodeRouter.HandleFunc("/message", MessageHandler(t.Context(), t)).Methods("POST")
	nodeRouter.HandleFunc("/ballot", BallotHandler(t.Context(), t)).Methods("POST")
	// nodeRouter.HandleFunc("/transactions", TransactionstHandler(t.Context(), t)).Methods("POST")

	t.server.Handler = handlers.CombinedLoggingHandler(t.config.HTTP2LogOutput, t.router)

	t.ready = true

	return nil
}

func (t *HTTP2Network) IsReady() bool {
	client, err := sebakcommon.NewHTTP2Client(50*time.Millisecond, 50*time.Millisecond, false)
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

func (t *HTTP2Network) Start() (err error) {
	defer func() {
		close(t.receiveChannel)
	}()

	return t.server.ListenAndServeTLS(t.tlsCertFile, t.tlsKeyFile)
}

func (t *HTTP2Network) Stop() {
	t.server.Close()
}

func (t *HTTP2Network) ReceiveChannel() chan Message {
	return t.receiveChannel
}

func (t *HTTP2Network) ReceiveMessage() <-chan Message {
	return t.receiveChannel
}
