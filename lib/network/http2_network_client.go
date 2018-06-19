package sebaknetwork

import (
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"boscoin.io/sebak/lib/common"
)

type HTTP2NetworkClient struct {
	endpoint       *sebakcommon.Endpoint
	client         *sebakcommon.HTTP2Client
	defaultHeaders http.Header
}

var (
	defaultTimeout     = 3 * time.Second
	defaultIdleTimeout = 3 * time.Second
)

func NewHTTP2NetworkClient(endpoint *sebakcommon.Endpoint, client *sebakcommon.HTTP2Client) *HTTP2NetworkClient {
	if client == nil {
		client, _ = sebakcommon.NewHTTP2Client(
			defaultTimeout,
			defaultIdleTimeout,
			false,
		)
	}

	return &HTTP2NetworkClient{endpoint: endpoint, client: client, defaultHeaders: http.Header{}}
}

func (c *HTTP2NetworkClient) Endpoint() *sebakcommon.Endpoint {
	return c.endpoint
}

func (c *HTTP2NetworkClient) SetDefaultHeaders(headers http.Header) {
	for key, values := range headers {
		for _, v := range values {
			c.defaultHeaders.Set(key, v)
		}
	}
}

func (c *HTTP2NetworkClient) DefaultHeaders() http.Header {
	headers := http.Header{}
	for key, values := range c.defaultHeaders {
		for _, v := range values {
			headers.Set(key, v)
		}
	}

	return headers
}

func (c *HTTP2NetworkClient) resolvePath(path string) (u *url.URL) {
	u = (*url.URL)(c.endpoint).ResolveReference(&url.URL{Path: path})
	return u
}

func (c *HTTP2NetworkClient) GetNodeInfo() (body []byte, err error) {
	headers := c.DefaultHeaders()
	headers.Set("Content-Type", "application/json")

	u := c.resolvePath("/get-node-info")

	var response *http.Response
	response, err = c.client.Get(u.String(), headers)
	if err != nil {
		return
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusOK {
		body, err = ioutil.ReadAll(response.Body)
	}
	return
}

func (c *HTTP2NetworkClient) Connect(node sebakcommon.Node) (body []byte, err error) {
	headers := c.DefaultHeaders()
	headers.Set("Content-Type", "application/json")

	n, _ := node.Serialize()
	var response *http.Response
	response, err = c.client.Post(c.resolvePath("/connect").String(), n, headers)
	if err != nil {
		return
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusOK {
		body, err = ioutil.ReadAll(response.Body)
	}
	return
}

func (c *HTTP2NetworkClient) SendMessage(message sebakcommon.Serializable) (retBody []byte, err error) {
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
	if response.StatusCode == http.StatusOK {
		retBody, err = ioutil.ReadAll(response.Body)
	}

	return
}

func (c *HTTP2NetworkClient) SendBallot(message sebakcommon.Serializable) (retBody []byte, err error) {
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
	if response.StatusCode == http.StatusOK {
		retBody, err = ioutil.ReadAll(response.Body)
	}

	return
}
