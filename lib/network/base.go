package network

import "net/url"

type Transport interface {
	Start() error
	Ready() error
	Send([]byte) error
	ReceiveChannel() chan TransportMessage
	ReceiveMessage() <-chan TransportMessage

	Endpoint() *url.URL
}
