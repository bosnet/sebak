package httpcache

import "net/http"

type NopClient struct {
}

func (NopClient) WrapHandlerFunc(handlerFunc http.HandlerFunc) http.HandlerFunc {
	return handlerFunc
}

func NewNopClient() *NopClient {
	return &NopClient{}
}
