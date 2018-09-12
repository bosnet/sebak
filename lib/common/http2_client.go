package common

import (
	"bytes"
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"time"

	"golang.org/x/net/http2"
)

type HTTP2Client struct {
	client    http.Client
	transport *http.Transport
}

func NewHTTP2Client(timeout, idleTimeout time.Duration, keepAlive bool) (client *HTTP2Client, err error) {
	if keepAlive {
		timeout, idleTimeout = 0, 0
	}

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
		IdleConnTimeout:   idleTimeout,
		DisableKeepAlives: !keepAlive,
		DialContext: (&net.Dialer{
			Timeout:   1 * time.Second,
			KeepAlive: 100000 * time.Second,
			DualStack: true,
		}).DialContext,
	}

	if err = http2.ConfigureTransport(transport); err != nil {
		return
	}

	client = &HTTP2Client{
		client: http.Client{
			Transport: transport,
			Timeout:   timeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse // NOTE prevent redirect
			},
		},
		transport: transport,
	}

	return
}

func (c *HTTP2Client) Close() {
	c.transport.CloseIdleConnections()
}

func (c *HTTP2Client) Get(url string, headers http.Header) (response *http.Response, err error) {
	var request *http.Request
	if request, err = http.NewRequest("GET", url, nil); err != nil {
		return
	}
	request.Header = headers

	if response, err = c.client.Do(request); err != nil {
		return
	}

	return
}

func (c *HTTP2Client) Post(url string, b []byte, headers http.Header) (response *http.Response, err error) {
	var request *http.Request
	if request, err = http.NewRequest("POST", url, bytes.NewBuffer(b)); err != nil {
		return
	}
	request.Header = headers

	if response, err = c.client.Do(request); err != nil {
		return
	}
	return
}

type HTTP2StreamWriter struct {
	DataChannel chan []byte
	Error       error

	DataChannelClosed bool
}

func NewHTTP2StreamWriter() *HTTP2StreamWriter {
	return &HTTP2StreamWriter{
		DataChannel: make(chan []byte),
	}
}

func (r *HTTP2StreamWriter) Write(b []byte) (int, error) {
	if r.DataChannelClosed {
		return 0, nil
	}

	r.DataChannel <- b
	return len(b), nil
}

func (r *HTTP2StreamWriter) Close() error {
	r.DataChannelClosed = true
	close(r.DataChannel)

	return nil
}

func GetHTTP2Stream(response *http.Response) (cw *HTTP2StreamWriter, err error) {
	cw = NewHTTP2StreamWriter()
	go func() {
		defer func() {
			response.Body.Close()
			cw.Close()
		}()

		_, err := io.Copy(cw, response.Body)
		if err != nil {
			cw.Error = err // maybe `io.ErrUnexpectedEOF`; connection lost
		}
	}()

	return
}
