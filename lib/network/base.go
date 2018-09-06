package network

import (
	"encoding/json"
	"io"
	"math"
	"net"
	"net/http"

	"github.com/gorilla/mux"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/node"
)

const (
	ConnectMessage                 = "connect"
	TransactionMessage MessageType = "transaction"
	BallotMessage                  = "ballot"
)

type Network interface {
	Endpoint() *common.Endpoint
	GetClient(endpoint *common.Endpoint) NetworkClient
	AddWatcher(func(Network, net.Conn, http.ConnState))
	AddHandler(string, http.HandlerFunc) *mux.Route

	Start() error
	Stop()
	SetMessageBroker(MessageBroker)
	MessageBroker() MessageBroker
	Ready() error
	IsReady() bool

	ReceiveChannel() chan Message
	ReceiveMessage() <-chan Message
}

func NewNetwork(endpoint *common.Endpoint) (n Network, err error) {
	switch endpoint.Scheme {
	case "memory":
		n = NewMemoryNetwork()
	case "https", "http":
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
	Endpoint() *common.Endpoint

	Connect(node node.Node) ([]byte, error)
	GetNodeInfo() ([]byte, error)
	SendMessage(common.Serializable) ([]byte, error)
	SendBallot(common.Serializable) ([]byte, error)
}

type MessageType string

func (t MessageType) String() string {
	return string(t)
}

// TODO versioning

type Message struct {
	Type MessageType
	Data []byte
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
	}
}

type MessageBroker interface {
	Response(io.Writer, []byte) error
	Receive(Message)
}
