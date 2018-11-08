package httpcache

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"strings"
	"time"

	logging "github.com/inconshreveable/log15"

	"boscoin.io/sebak/lib/common"
)

type Client struct {
	adapter     Adapter
	ttl         time.Duration
	methods     map[string]bool
	statusCodes map[int]time.Duration
	logger      logging.Logger
}

type ClientOption func(c *Client) error

func NewClient(opts ...ClientOption) (*Client, error) {
	c := &Client{
		methods:     map[string]bool{"GET": true},
		statusCodes: map[int]time.Duration{},
		ttl:         time.Duration(0),
		logger:      common.NopLogger(),
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
		c.ttl = ttl
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

func WithOptions(options ...ClientOption) ClientOption {
	return func(c *Client) error {
		for _, opt := range options {
			if err := opt(c); err != nil {
				return err
			}
		}
		return nil
	}
}

func WithStatusCode(code int, ttl time.Duration) ClientOption {
	return func(c *Client) error {
		c.statusCodes[code] = ttl
		return nil
	}
}

func WithLogger(logger logging.Logger) ClientOption {
	return func(c *Client) error {
		c.logger = logger
		return nil
	}
}

func (c *Client) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ok := c.handleCache(next, w, r); !ok {
			c.logger.Debug("page not cached", "url", r.URL.String())
			next.ServeHTTP(w, r)
		}
	})
}

func (c *Client) WrapHandlerFunc(handlerFunc http.HandlerFunc) http.HandlerFunc {
	next := http.HandlerFunc(handlerFunc)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ok := c.handleCache(next, w, r); !ok {
			c.logger.Debug("page not cached", "url", r.URL.String())
			next.ServeHTTP(w, r)
		}
	})

}

func (c *Client) handleCache(next http.Handler, w http.ResponseWriter, r *http.Request) bool {
	if ok := c.methods[r.Method]; !ok {
		return false
	}
	sortURLParams(r.URL)
	key := r.URL.String()
	resp, ok := c.adapter.Get(key)
	if ok {
		if resp.Expiration.IsZero() || resp.Expiration.After(time.Now()) {
			for k, v := range resp.Header {
				w.Header().Set(k, strings.Join(v, ","))
			}
			w.WriteHeader(resp.StatusCode)
			w.Write(resp.Value)
			c.logger.Debug("return cache", "url", r.URL.String())
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
		c.logger.Debug("page cached", "url", r.URL.String(), "code", statusCode, "expir", expiration)
	}
	for k, v := range result.Header {
		w.Header().Set(k, strings.Join(v, ","))
	}
	w.WriteHeader(statusCode)
	w.Write(value)
	return true
}

func (c *Client) cachingExpiration(code int) (time.Time, bool) {
	if ttl, ok := c.statusCodes[code]; ok {
		return expiration(ttl), true
	} else if code < 400 {
		return expiration(c.ttl), true
	}
	return time.Time{}, false
}

func expiration(ttl time.Duration) time.Time {
	if ttl == 0 {
		return time.Time{}
	}
	return time.Now().Add(ttl)
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
