package network

import (
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/spikeekips/sebak/lib/util"
)

type HTTP2TransportClient struct {
	endpoint *util.Endpoint
	client   *util.HTTP2Client
}

func NewHTTP2TransportClient(endpoint *util.Endpoint, client *util.HTTP2Client) *HTTP2TransportClient {
	return &HTTP2TransportClient{endpoint: endpoint, client: client}
}

func (c *HTTP2TransportClient) Endpoint() *util.Endpoint {
	return c.endpoint
}

func (c *HTTP2TransportClient) resolvePath(path string) (u *url.URL) {
	u = (*url.URL)(c.endpoint).ResolveReference(&url.URL{Path: path})
	return u
}

func (c *HTTP2TransportClient) GetNodeInfo() (body []byte, err error) {
	headers := http.Header{}
	headers.Set("Content-Type", "application/json")

	u := c.resolvePath("/get-node-info/")

	var response *http.Response
	response, err = c.client.Get(u.String(), headers)
	defer response.Body.Close()
	if err != nil {
		return
	}
	body, err = ioutil.ReadAll(response.Body)
	return
}

func (c *HTTP2TransportClient) SendMessage(message util.Serializable) (err error) {
	headers := http.Header{}
	headers.Set("Content-Type", "application/json")

	var body []byte
	if body, err = message.Serialize(); err != nil {
		return
	}

	u := c.resolvePath("/message")

	var response *http.Response
	response, err = c.client.Post(u.String(), body, headers)
	defer response.Body.Close()
	if err != nil {
		return
	}

	return
}

func (c *HTTP2TransportClient) SendBallot(message util.Serializable) (err error) {
	headers := http.Header{}
	headers.Set("Content-Type", "application/json")

	var body []byte
	if body, err = message.Serialize(); err != nil {
		return
	}

	u := c.resolvePath("/ballot")

	var response *http.Response
	response, err = c.client.Post(u.String(), body, headers)
	defer response.Body.Close()
	if err != nil {
		return
	}

	return
}
