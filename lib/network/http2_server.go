package network

import (
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/spikeekips/sebak/lib"
	"github.com/spikeekips/sebak/lib/util"
	"golang.org/x/net/http2"
)

type TransportMessage struct {
	Type string
	Data []byte
}

type HTTP2Transport struct {
	tlsCertFile string
	tlsKeyFile  string

	server *http.Server

	receiveChannel chan TransportMessage
	receiveQueue   *util.Queue

	ready bool

	handlers map[string]func(http.ResponseWriter, *http.Request)
	watchers []func(*HTTP2Transport, net.Conn, http.ConnState)
}

type HTTP2TransportConfig struct {
	Addr string

	ReadTimeout,
	ReadHeaderTimeout,
	WriteTimeout,
	IdleTimeout time.Duration

	TlsCertFile,
	TlsKeyFile string
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
		tlsCertFile:    config.TlsCertFile,
		tlsKeyFile:     config.TlsKeyFile,
		receiveChannel: make(chan TransportMessage),
		receiveQueue:   &util.Queue{},
	}

	transport.handlers = map[string]func(http.ResponseWriter, *http.Request){}

	transport.setNotReadyHandler()
	transport.server.ConnState = transport.ConnState

	return transport
}

func (t *HTTP2Transport) Endpoint() string {
	return fmt.Sprintf("https://%s", t.server.Addr)
}

func (t *HTTP2Transport) AddWatcher(f func(*HTTP2Transport, net.Conn, http.ConnState)) {
	t.watchers = append(t.watchers, f)
}

func (t *HTTP2Transport) ConnState(c net.Conn, state http.ConnState) {
	for _, f := range t.watchers {
		f(t, c, state)
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

	t.server.Handler = handler
}

func (t *HTTP2Transport) AddHandler(pattern string, handler func(*HTTP2Transport) HandlerFunc) (err error) {
	t.handlers[pattern] = handler(t)
	return nil
}

func (t *HTTP2Transport) Ready() {
	handler := new(http.ServeMux)
	for pattern, handlerFunc := range t.handlers {
		handler.HandleFunc(pattern, handlerFunc)
	}
	t.server.Handler = handler

	t.ready = true
}

func (t *HTTP2Transport) Start() (err error) {
	defer func() {
		close(t.receiveChannel)
	}()

	return t.server.ListenAndServeTLS(t.tlsCertFile, t.tlsKeyFile)
}

func (t *HTTP2Transport) Receive() chan TransportMessage {
	return t.receiveChannel
}

func (t *HTTP2Transport) Send(node sebak.Node, b []byte) (err error) {
	return nil
}

func (t *HTTP2Transport) SendRaw(endpoint string, b []byte) (err error) {
	return nil
}
