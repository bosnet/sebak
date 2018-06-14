package sebakcommon

import (
	"errors"
	"net"
	"net/url"
	"strconv"
)

func CheckPortInUse(port int) error {
	_, err := net.DialTimeout("tcp", net.JoinHostPort("", strconv.FormatInt(int64(port), 10)), 10)
	return err
}

func CheckBindString(b string) error {
	_, port, err := net.SplitHostPort(b)
	if err != nil {
		return err
	}

	var portInt int64
	if portInt, err = strconv.ParseInt(port, 10, 64); err != nil {
		return err
	} else if portInt < 1 {
		return errors.New("invalid port")
	}

	return nil
}

type Endpoint url.URL

func NewEndpointFromURL(u *url.URL) *Endpoint {
	return (*Endpoint)(u)
}

func NewEndpointFromString(s string) (e *Endpoint, err error) {
	var u *url.URL
	if u, err = url.Parse(s); err != nil {
		return
	}
	e = NewEndpointFromURL(u)
	return
}

func (e *Endpoint) String() string {
	return (&url.URL{
		Scheme: e.Scheme,
		Host:   e.Host,
		Path:   e.Path,
	}).String()
}

func (e *Endpoint) IsRemote() bool {
	return e.Scheme != "memory"
}

func (e *Endpoint) HostAndPort() (string, string) {
	host, port, _ := net.SplitHostPort(e.Host)
	return host, port
}

func (e *Endpoint) Query() url.Values {
	return (*url.URL)(e).Query()
}

func (e *Endpoint) UnmarshalJSON(b []byte) error {
	p, err := ParseNodeEndpoint(string(b)[1 : len(string(b))-1])
	if err != nil {
		return err
	}

	*e = *p

	return nil
}
