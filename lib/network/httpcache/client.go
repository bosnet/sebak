package httpcache

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"strings"
	"time"
)

type Client struct {
	adapter     Adapter
	ttl         *time.Duration
	methods     map[string]bool
	statusCodes map[int]*time.Duration
}

type ClientOption func(c *Client) error

func NewClient(opts ...ClientOption) (*Client, error) {
	c := &Client{
		methods:     map[string]bool{"GET": true},
		statusCodes: map[int]*time.Duration{},
		ttl:         nil,
	}

	for _, opt := range opts {
		if err := opt(c); err != nil {
			return nil, err
		}
	}

	if c.adapter == nil {
		return nil, errors.New("cache client adapter is nil")
	}

	return c, nil
}

func WithAdapter(a Adapter) ClientOption {
	return func(c *Client) error {
		c.adapter = a
		return nil
	}
}

func WithExpire(ttl time.Duration) ClientOption {
	return func(c *Client) error {
		c.ttl = &ttl
		return nil
	}
}

func WithMethods(methods ...string) ClientOption {
	return func(c *Client) error {
		for _, m := range methods {
			c.methods[m] = true
		}
		return nil
	}
}

func WithStatusCode(code int, ttl time.Duration) ClientOption {
	return func(c *Client) error {
		c.statusCodes[code] = &ttl
		return nil
	}
}

func (c *Client) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ok := c.handleCache(next, w, r); !ok {
			next.ServeHTTP(w, r)
		}
	})
}

func (c *Client) handleCache(next http.Handler, w http.ResponseWriter, r *http.Request) bool {
	if ok := c.methods[r.Method]; ok {
		sortURLParams(r.URL)
		key := r.URL.String()
		resp, ok := c.adapter.Get(key)
		if ok {
			if resp.Expiration == nil || resp.Expiration.After(time.Now()) {
				for k, v := range resp.Header {
					w.Header().Set(k, strings.Join(v, ","))
				}
				w.WriteHeader(resp.StatusCode)
				w.Write(resp.Value)
				return true
			}
			c.adapter.Remove(key)
		}
		rec := httptest.NewRecorder()
		next.ServeHTTP(rec, r)
		var (
			result              = rec.Result()
			statusCode          = result.StatusCode
			value               = rec.Body.Bytes()
			expiration, caching = c.cachingExpiration(statusCode)
		)
		if caching {
			resp := &Response{
				Value:      value,
				StatusCode: statusCode,
				Header:     result.Header,
				Expiration: expiration,
			}
			c.adapter.Set(key, resp, expiration)
		}
		for k, v := range result.Header {
			w.Header().Set(k, strings.Join(v, ","))
		}
		w.WriteHeader(statusCode)
		w.Write(value)
		return true
	}
	return false
}

func (c *Client) cachingExpiration(code int) (*time.Time, bool) {
	if ttl, ok := c.statusCodes[code]; ok {
		return expiration(ttl), true
	} else if code < 400 {
		return expiration(c.ttl), true
	}
	return nil, false
}

func expiration(ttl *time.Duration) *time.Time {
	if ttl != nil {
		t := time.Now().Add(*ttl)
		return &t
	}
	return nil
}

func sortURLParams(u *url.URL) {
	params := u.Query()
	for _, p := range params {
		sort.Slice(p, func(i, j int) bool {
			return p[i] < p[j]
		})
	}
	u.RawQuery = params.Encode()
}
