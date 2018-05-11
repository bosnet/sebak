package network

import (
	"encoding/json"
	"net/url"
)

type Transport interface {
	Start() error
	Ready() error
	Send(TransportMessageType, []byte) error

	ReceiveChannel() chan TransportMessage
	ReceiveMessage() <-chan TransportMessage

	Endpoint() *url.URL
}

type TransportMessageType string

func (t TransportMessageType) String() string {
	return string(t)
}

const (
	MessageTransportMessage     TransportMessageType = "message"
	BallotTransportMessage                           = "ballot"
	GetNodeInfoTransportMessage                      = "get-node-info"
)

type TransportMessage struct {
	Type       TransportMessageType
	Data       []byte
	DataString string // optional
}

func (t TransportMessage) String() string {
	o, _ := json.MarshalIndent(t, "", "  ")
	return string(o)
}

func NewTransportMessage(mt TransportMessageType, data []byte) TransportMessage {
	return TransportMessage{
		Type:       mt,
		Data:       data,
		DataString: string(data),
	}
}

// TODO rename `network.Transport` to `network.Network`
func NewNetwork(endpoint *url.URL) (transport Transport, err error) {
	switch endpoint.Scheme {
	case "memory":
		transport = NewMemNetwork()
	case "https":
		var config HTTP2TransportConfig
		config, err = NewHTTP2TransportConfigFromEndpoint(endpoint)
		if err != nil {
			return
		}
		transport = NewHTTP2Transport(config)
	}

	return
}
