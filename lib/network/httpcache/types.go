package httpcache

import (
	"net/http"
	"time"
)

type Adapter interface {
	Get(key string) (*Response, bool)
	Set(key string, response *Response, expiration *time.Time)
	Remove(key string)
}

type Response struct {
	Value      []byte
	Header     http.Header
	Expiration *time.Time
}
