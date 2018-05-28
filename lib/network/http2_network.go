package sebaknetwork

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/handlers"
	"github.com/spikeekips/sebak/lib/common"

	"golang.org/x/net/http2"
)

type HTTP2NetworkConfig struct {
	NodeName string
	Addr     string

	ReadTimeout,
	ReadHeaderTimeout,
	WriteTimeout,
	IdleTimeout time.Duration

	TLSCertFile,
	TLSKeyFile string

	HTTP2LogOutput io.Writer
}

func NewHTTP2NetworkConfigFromEndpoint(endpoint *sebakcommon.Endpoint) (config HTTP2NetworkConfig, err error) {
	query := endpoint.Query()

	var NodeName string
	var ReadTimeout time.Duration = 0
	var ReadHeaderTimeout time.Duration = 0
	var WriteTimeout time.Duration = 0
	var IdleTimeout time.Duration = 5
	var TLSCertFile, TLSKeyFile string
	var HTTP2LogOutput io.Writer

	if ReadTimeout, err = time.ParseDuration(sebakcommon.GetUrlQuery(query, "ReadTimeout", "0s")); err != nil {
		return
	}
	if ReadTimeout < 0*time.Second {
		err = errors.New("invalid 'ReadTimeout'")
		return
	}

	if ReadHeaderTimeout, err = time.ParseDuration(sebakcommon.GetUrlQuery(query, "ReadHeaderTimeout", "0s")); err != nil {
		return
	}
	if ReadHeaderTimeout < 0*time.Second {
		err = errors.New("invalid 'ReadHeaderTimeout'")
		return
	}

	if WriteTimeout, err = time.ParseDuration(sebakcommon.GetUrlQuery(query, "WriteTimeout", "0s")); err != nil {
		return
	}
	if WriteTimeout < 0*time.Second {
		err = errors.New("invalid 'WriteTimeout'")
		return
	}

	if IdleTimeout, err = time.ParseDuration(sebakcommon.GetUrlQuery(query, "IdleTimeout", "0s")); err != nil {
		return
	}
	if IdleTimeout < 0*time.Second {
		err = errors.New("invalid 'IdleTimeout'")
		return
	}

	if v := query.Get("TLSCertFile"); len(v) < 1 {
		err = errors.New("'TLSCertFile' is missing")
		return
	} else {
		TLSCertFile = v
	}

	if v := query.Get("TLSKeyFile"); len(v) < 1 {
		err = errors.New("'TLSKeyFile' is missing")
		return
	} else {
		TLSKeyFile = v
	}

	if v := query.Get("NodeName"); len(v) < 1 {
		err = errors.New("`NodeName` must be given")
		return
	} else {
		NodeName = v
	}

	if v := query.Get("HTTP2LogOutput"); len(v) < 1 {
		HTTP2LogOutput = os.Stdout
	} else {
		HTTP2LogOutput, err = os.OpenFile(v, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return
		}
	}

	config = HTTP2NetworkConfig{
		NodeName:          NodeName,
		Addr:              endpoint.Host,
		ReadTimeout:       ReadTimeout,
		ReadHeaderTimeout: ReadHeaderTimeout,
		WriteTimeout:      WriteTimeout,
		IdleTimeout:       IdleTimeout,
		TLSCertFile:       TLSCertFile,
		TLSKeyFile:        TLSKeyFile,
		HTTP2LogOutput:    HTTP2LogOutput,
	}

	return
}

type HTTP2Network struct {
	ctx         context.Context
	tlsCertFile string
	tlsKeyFile  string

	server *http.Server

	receiveChannel chan Message

	ready bool

	handlers map[string]func(http.ResponseWriter, *http.Request)
	watchers []func(Network, net.Conn, http.ConnState)

	config HTTP2NetworkConfig
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

	h2n = &HTTP2Network{
		server:         server,
		tlsCertFile:    config.TLSCertFile,
		tlsKeyFile:     config.TLSKeyFile,
		receiveChannel: make(chan Message),
	}

	h2n.config = config
	h2n.handlers = map[string]func(http.ResponseWriter, *http.Request){}

	h2n.setNotReadyHandler()
	h2n.server.ConnState = h2n.ConnState

	return
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
	handler := http.NewServeMux()
	handler.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if !t.ready {
			http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
			return
		}
	})

	t.server.Handler = handlers.CombinedLoggingHandler(t.config.HTTP2LogOutput, handler)
}

func (t *HTTP2Network) AddHandler(ctx context.Context, pattern string, handler func(context.Context, *HTTP2Network) HandlerFunc) (err error) {
	t.handlers[pattern] = handler(ctx, t)
	return nil
}

func (t *HTTP2Network) Ready() error {
	t.AddHandler(t.Context(), "/", Index)
	t.AddHandler(t.Context(), "/connect", ConnectHandler)
	t.AddHandler(t.Context(), "/message", MessageHandler)
	t.AddHandler(t.Context(), "/ballot", BallotHandler)

	handler := new(http.ServeMux)
	for pattern, handlerFunc := range t.handlers {
		handler.HandleFunc(pattern, handlerFunc)
	}

	t.server.Handler = handlers.CombinedLoggingHandler(t.config.HTTP2LogOutput, handler)

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
