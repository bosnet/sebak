package httpcache

import (
	"net/http"
	"time"
)

type Adapter interface {
	Get(key string) (*Response, bool)
	Set(key string, response *Response, expiration time.Time)
	Remove(key string)
}

type Response struct {
	Value      []byte
	StatusCode int
	Header     http.Header
	Expiration time.Time
}

type Wrapper interface {
	WrapHandlerFunc(handlerFunc http.HandlerFunc) http.HandlerFunc
}
