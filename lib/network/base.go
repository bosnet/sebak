package network

import (
	"context"
	"encoding/json"
	"time"

	"github.com/spikeekips/sebak/lib/util"
)

type TransportServer interface {
	Endpoint() *util.Endpoint
	Context() context.Context
	SetContext(context.Context)

	Start() error
	Ready() error

	ReceiveChannel() chan Message
	ReceiveMessage() <-chan Message
}

type TransportClient interface {
	Endpoint() *util.Endpoint
	Timeout() time.Duration

	GetNodeInfo() ([]byte, error)
	SendMessage(util.Serializable) error
	SendBallot(util.Serializable) error
}

type MessageType string

func (t MessageType) String() string {
	return string(t)
}

const (
	MessageFromClient  MessageType = "message"
	BallotMessage                  = "ballot"
	GetNodeInfoMessage             = "get-node-info"
)

type Message struct {
	Type       MessageType
	Data       []byte
	DataString string // optional
}

func (t Message) String() string {
	o, _ := json.MarshalIndent(t, "", "  ")
	return string(o)
}

func NewMessage(mt MessageType, data []byte) Message {
	return Message{
		Type:       mt,
		Data:       data,
		DataString: string(data),
	}
}

func NewTransportServer(endpoint *util.Endpoint) (transport TransportServer, err error) {
	switch endpoint.Scheme {
	case "memory":
		transport = NewMemoryTransport()
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
