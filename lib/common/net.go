package common

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/ulule/limiter"
)

var DefaultEndpoint int = 12345

func CheckPortInUse(port int) error {
	if port < 1 {
		return errors.New("0 port is not available")
	}
	_, err := net.DialTimeout(
		"tcp",
		net.JoinHostPort("", strconv.FormatInt(int64(port), 10)),
		100*time.Millisecond,
	)
	return err
}

func GetFreePort(excludes ...int) (port int) {
	for i := 1024; i < 10000; i++ {
		var found bool
		for _, e := range excludes {
			if i == e {
				found = true
				break
			}
		}
		if found {
			continue
		}

		if err := CheckPortInUse(i); err == nil {
			continue
		}
		port = i
		break
	}

	return
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

	u.Scheme = strings.ToLower(u.Scheme)
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

func (e *Endpoint) Query() url.Values {
	return (*url.URL)(e).Query()
}

func (e *Endpoint) Port() string {
	return (*url.URL)(e).Port()
}

func (e *Endpoint) UnmarshalJSON(b []byte) error {
	p, err := ParseEndpoint(string(b)[1 : len(string(b))-1])
	if err != nil {
		return err
	}

	*e = *p

	return nil
}

func ParseEndpoint(endpoint string) (u *Endpoint, err error) {
	var parsed *url.URL
	parsed, err = url.Parse(endpoint)
	if err != nil {
		return
	}
	if len(parsed.Scheme) < 1 {
		err = errors.New("missing scheme")
		return
	}

	if len(parsed.Port()) < 1 && parsed.Scheme != "memory" {
		parsed.Host = fmt.Sprintf("%s:%d", parsed.Host, DefaultEndpoint)
	}

	if parsed.Scheme != "memory" {
		var port string
		port = parsed.Port()

		var portInt int64
		if portInt, err = strconv.ParseInt(port, 10, 64); err != nil {
			return
		} else if portInt < 1 {
			err = errors.New("invalid port")
			return
		}

		if len(parsed.Host) < 1 || strings.HasPrefix(parsed.Host, "127.0.") {
			parsed.Host = fmt.Sprintf("localhost:%s", parsed.Port())
		}
	}

	parsed.Host = strings.ToLower(parsed.Host)

	u = (*Endpoint)(parsed)

	return
}

func RequestURLFromRequest(r *http.Request) *url.URL {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}

	return &url.URL{
		Scheme:     scheme,
		Opaque:     r.URL.Opaque,
		User:       r.URL.User,
		Host:       r.Host,
		Path:       r.URL.Path,
		RawPath:    r.URL.RawPath,
		ForceQuery: r.URL.ForceQuery,
		RawQuery:   r.URL.RawQuery,
		Fragment:   r.URL.Fragment,
	}
}

type RateLimitRule struct {
	Default     limiter.Rate
	ByIPAddress map[string]limiter.Rate
}

func NewRateLimitRule(rate limiter.Rate) RateLimitRule {
	return RateLimitRule{
		Default:     rate,
		ByIPAddress: map[string]limiter.Rate{},
	}
}

func (r RateLimitRule) Serializable() ([]byte, error) {
	return json.Marshal(r)
}
