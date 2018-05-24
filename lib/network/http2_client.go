package network

import (
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/spikeekips/sebak/lib/util"
)

type HTTP2TransportClient struct {
	endpoint       *util.Endpoint
	client         *util.HTTP2Client
	defaultHeaders http.Header
}

var (
	defaultTimeout     = 3 * time.Second
	defaultIdleTimeout = 3 * time.Second
)

func NewHTTP2TransportClient(endpoint *util.Endpoint, client *util.HTTP2Client) *HTTP2TransportClient {
	if client == nil {
		client, _ = util.NewHTTP2Client(
			defaultTimeout,
			defaultIdleTimeout,
			false,
		)
	}

	return &HTTP2TransportClient{endpoint: endpoint, client: client, defaultHeaders: http.Header{}}
}

func (c *HTTP2TransportClient) Endpoint() *util.Endpoint {
	return c.endpoint
}

func (c *HTTP2TransportClient) SetDefaultHeaders(headers http.Header) {
	for key, values := range headers {
		for _, v := range values {
			c.defaultHeaders.Set(key, v)
		}
	}
}

func (c *HTTP2TransportClient) DefaultHeaders() http.Header {
	headers := http.Header{}
	for key, values := range c.defaultHeaders {
		for _, v := range values {
			headers.Set(key, v)
		}
	}

	return headers
}

func (c *HTTP2TransportClient) resolvePath(path string) (u *url.URL) {
	u = (*url.URL)(c.endpoint).ResolveReference(&url.URL{Path: path})
	return u
}

func (c *HTTP2TransportClient) GetNodeInfo() (body []byte, err error) {
	headers := c.DefaultHeaders()
	headers.Set("Content-Type", "application/json")

	u := c.resolvePath("/get-node-info")

	var response *http.Response
	response, err = c.client.Get(u.String(), headers)
	if err != nil {
		return
	}
	defer response.Body.Close()
	body, err = ioutil.ReadAll(response.Body)
	return
}

func (c *HTTP2TransportClient) Connect(node util.Node) (body []byte, err error) {
	headers := c.DefaultHeaders()
	headers.Set("Content-Type", "application/json")

	n, _ := node.Serialize()
	var response *http.Response
	response, err = c.client.Post(c.resolvePath("/connect").String(), n, headers)
	if err != nil {
		return
	}
	defer response.Body.Close()
	body, err = ioutil.ReadAll(response.Body)
	return
}

func (c *HTTP2TransportClient) SendMessage(message util.Serializable) (err error) {
	headers := c.DefaultHeaders()
	headers.Set("Content-Type", "application/json")

	var body []byte
	if body, err = message.Serialize(); err != nil {
		return
	}

	u := c.resolvePath("/message")

	var response *http.Response
	response, err = c.client.Post(u.String(), body, headers)
	if err != nil {
		return
	}
	defer response.Body.Close()

	return
}

func (c *HTTP2TransportClient) SendBallot(message util.Serializable) (err error) {
	headers := c.DefaultHeaders()
	headers.Set("Content-Type", "application/json")

	var body []byte
	if body, err = message.Serialize(); err != nil {
		return
	}

	u := c.resolvePath("/ballot")

	var response *http.Response
	response, err = c.client.Post(u.String(), body, headers)
	if err != nil {
		return
	}
	defer response.Body.Close()

	return
}
