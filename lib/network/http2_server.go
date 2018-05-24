package network

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
	"github.com/spikeekips/sebak/lib/util"

	"golang.org/x/net/http2"
)

type HTTP2TransportConfig struct {
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

func NewHTTP2TransportConfigFromEndpoint(endpoint *util.Endpoint) (config HTTP2TransportConfig, err error) {
	query := endpoint.Query()

	var NodeName string
	var ReadTimeout time.Duration = 0
	var ReadHeaderTimeout time.Duration = 0
	var WriteTimeout time.Duration = 0
	var IdleTimeout time.Duration = 5
	var TLSCertFile, TLSKeyFile string
	var HTTP2LogOutput io.Writer

	if ReadTimeout, err = time.ParseDuration(util.GetUrlQuery(query, "ReadTimeout", "0s")); err != nil {
		return
	}
	if ReadTimeout < 0*time.Second {
		err = errors.New("invalid 'ReadTimeout'")
		return
	}

	if ReadHeaderTimeout, err = time.ParseDuration(util.GetUrlQuery(query, "ReadHeaderTimeout", "0s")); err != nil {
		return
	}
	if ReadHeaderTimeout < 0*time.Second {
		err = errors.New("invalid 'ReadHeaderTimeout'")
		return
	}

	if WriteTimeout, err = time.ParseDuration(util.GetUrlQuery(query, "WriteTimeout", "0s")); err != nil {
		return
	}
	if WriteTimeout < 0*time.Second {
		err = errors.New("invalid 'WriteTimeout'")
		return
	}

	if IdleTimeout, err = time.ParseDuration(util.GetUrlQuery(query, "IdleTimeout", "0s")); err != nil {
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

	config = HTTP2TransportConfig{
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

type HTTP2Transport struct {
	ctx         context.Context
	tlsCertFile string
	tlsKeyFile  string

	server *http.Server

	receiveChannel chan Message

	ready bool

	handlers map[string]func(http.ResponseWriter, *http.Request)
	watchers []func(TransportServer, net.Conn, http.ConnState)

	config HTTP2TransportConfig
}

type HandlerFunc func(w http.ResponseWriter, r *http.Request)

func NewHTTP2Transport(config HTTP2TransportConfig) (transport *HTTP2Transport) {
	server := &http.Server{
		Addr:              config.Addr,
		ReadTimeout:       config.ReadTimeout,
		ReadHeaderTimeout: config.ReadHeaderTimeout,
		WriteTimeout:      config.WriteTimeout,
		//ErrorLog: , // TODO replace custom logger
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

	transport = &HTTP2Transport{
		server:         server,
		tlsCertFile:    config.TLSCertFile,
		tlsKeyFile:     config.TLSKeyFile,
		receiveChannel: make(chan Message),
	}

	transport.config = config
	transport.handlers = map[string]func(http.ResponseWriter, *http.Request){}

	transport.setNotReadyHandler()
	transport.server.ConnState = transport.ConnState

	return transport
}

func (t *HTTP2Transport) Context() context.Context {
	return t.ctx
}

func (t *HTTP2Transport) SetContext(ctx context.Context) {
	t.ctx = ctx
}

// GetClient creates new keep-alive HTTP2 client
func (t *HTTP2Transport) GetClient(endpoint *util.Endpoint) TransportClient {
	rawClient, _ := util.NewHTTP2Client(defaultTimeout, 0, true)

	client := NewHTTP2TransportClient(endpoint, rawClient)

	headers := http.Header{}
	headers.Set("User-Agent", fmt.Sprintf("v-%s", t.config.NodeName))
	client.SetDefaultHeaders(headers)

	return client
}

func (t *HTTP2Transport) Endpoint() *util.Endpoint {
	host, port, _ := net.SplitHostPort(t.server.Addr)
	return &util.Endpoint{Scheme: "https", Host: fmt.Sprintf("%s:%s", host, port)}
}

func (t *HTTP2Transport) AddWatcher(f func(TransportServer, net.Conn, http.ConnState)) {
	t.watchers = append(t.watchers, f)
}

func (t *HTTP2Transport) ConnState(c net.Conn, state http.ConnState) {
	for _, f := range t.watchers {
		go f(t, c, state)
	}
}

func (t *HTTP2Transport) setNotReadyHandler() {
	handler := http.NewServeMux()
	handler.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if !t.ready {
			http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
			return
		}
	})

	t.server.Handler = handlers.CombinedLoggingHandler(t.config.HTTP2LogOutput, handler)
}

func (t *HTTP2Transport) AddHandler(ctx context.Context, pattern string, handler func(context.Context, *HTTP2Transport) HandlerFunc) (err error) {
	t.handlers[pattern] = handler(ctx, t)
	return nil
}

func (t *HTTP2Transport) Ready() error {
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

func (t *HTTP2Transport) IsReady() bool {
	client, err := util.NewHTTP2Client(50*time.Millisecond, 50*time.Millisecond, false)
	if err != nil {
		return false
	}
	defer client.Close()

	transport := NewHTTP2TransportClient(t.Endpoint(), client)
	if _, err := transport.GetNodeInfo(); err != nil {
		return false
	}

	return true
}

func (t *HTTP2Transport) Start() (err error) {
	defer func() {
		close(t.receiveChannel)
	}()

	return t.server.ListenAndServeTLS(t.tlsCertFile, t.tlsKeyFile)
}

func (t *HTTP2Transport) Stop() {
	t.server.Close()
}

func (t *HTTP2Transport) ReceiveChannel() chan Message {
	return t.receiveChannel
}

func (t *HTTP2Transport) ReceiveMessage() <-chan Message {
	return t.receiveChannel
}
