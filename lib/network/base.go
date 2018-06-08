package sebaknetwork

import (
	"context"
	"encoding/json"
	"math"
	"net"
	"net/http"

	"github.com/owlchain/sebak/lib/common"
)

type Network interface {
	Endpoint() *sebakcommon.Endpoint
	Context() context.Context
	SetContext(context.Context)
	GetClient(endpoint *sebakcommon.Endpoint) NetworkClient
	AddWatcher(func(Network, net.Conn, http.ConnState))

	Start() error
	Stop()
	Ready() error
	IsReady() bool

	ReceiveChannel() chan Message
	ReceiveMessage() <-chan Message
}

func NewNetwork(endpoint *sebakcommon.Endpoint) (n Network, err error) {
	switch endpoint.Scheme {
	case "memory":
		n = NewMemoryNetwork()
	case "https":
		var config HTTP2NetworkConfig
		config, err = NewHTTP2NetworkConfigFromEndpoint(endpoint)
		if err != nil {
			return
		}
		n = NewHTTP2Network(config)
	}

	return
}

type NetworkClient interface {
	Endpoint() *sebakcommon.Endpoint

	Connect(node sebakcommon.Node) ([]byte, error)
	GetNodeInfo() ([]byte, error)
	SendMessage(sebakcommon.Serializable) error
	SendBallot(sebakcommon.Serializable) error
}

type MessageType string

func (t MessageType) String() string {
	return string(t)
}

const (
	MessageFromClient  MessageType = "message"
	ConnectMessage                 = "connect"
	BallotMessage                  = "ballot"
	GetNodeInfoMessage             = "get-node-info"
)

// TODO versioning

type Message struct {
	Type MessageType
	Data []byte
	//DataString string // optional
}

func (t Message) String() string {
	o, _ := json.Marshal(t)
	return string(o)
}

func (t Message) Head(n int) string {
	s := t.String()
	i := math.Min(
		float64(len(s)),
		float64(n),
	)
	return s[:int(i)]
}

func (t Message) IsEmpty() bool {
	return len(t.Data) < 1
}

func NewMessage(mt MessageType, data []byte) Message {
	return Message{
		Type: mt,
		Data: data,
		//DataString: string(data),
	}
}
