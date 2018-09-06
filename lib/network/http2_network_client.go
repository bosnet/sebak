package network

import (
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/node"
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

	u := c.resolvePath(UrlPathPrefixNode + "/")

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

func (c *HTTP2NetworkClient) Connect(n node.Node) (body []byte, err error) {
	headers := c.DefaultHeaders()
	headers.Set("Content-Type", "application/json")

	serialized, _ := n.Serialize()
	var response *http.Response
	response, err = c.client.Post(c.resolvePath(UrlPathPrefixNode+"/connect").String(), serialized, headers)
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

	u := c.resolvePath(UrlPathPrefixNode + "/message")

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

	u := c.resolvePath(UrlPathPrefixNode + "/ballot")

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

///
/// Perform a raw Get request on this peer
///
/// This is a quick way to request the API.
/// As APIs are rapidly evolving, wrapping all of them properly
/// would be counter productive, to this function is provided.
///
/// Params:
///   endpoint = URL chunk to request (e.g. `/api/foo?bar=baguette`)
///
/// Returns:
///   []byte = Body part returned by the query if it was successful
///   error  = Error information if the query wasn't successful
///
func (client *HTTP2NetworkClient) Get(endpoint string) ([]byte, error) {
	var body []byte
	var err error
	var response *http.Response
	headers := client.DefaultHeaders()

	headers.Set("Accept", "application/json")
	u := client.resolvePath(endpoint)

	if response, err = client.client.Get(u.String(), headers); err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusOK {
		body, err = ioutil.ReadAll(response.Body)
	}

	return body, err
}
