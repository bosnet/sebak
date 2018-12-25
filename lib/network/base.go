package network

import (
	"io"
	"net/http"

	"github.com/gorilla/mux"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/node"
)

type Network interface {
	Endpoint() *common.Endpoint
	GetClient(endpoint *common.Endpoint) NetworkClient
	AddHandler(string, http.HandlerFunc) *mux.Route
	AddMiddleware(string, ...mux.MiddlewareFunc) error

	// Starts network handling
	// Blocks until finished, either because of an error
	// or because `Stop` was called
	Start() error
	Stop()
	SetMessageBroker(MessageBroker)
	MessageBroker() MessageBroker
	Ready() error
	IsReady() bool

	ReceiveChannel() chan common.NetworkMessage
	ReceiveMessage() <-chan common.NetworkMessage
}

type NetworkClient interface {
	Endpoint() *common.Endpoint

	Connect(node node.Node) ([]byte, error)
	GetNodeInfo() ([]byte, error)
	SendMessage(interface{}) ([]byte, error)
	SendTransaction(interface{}) ([]byte, error)
	SendBallot(interface{}) ([]byte, error)
	SendDiscovery(interface{}) ([]byte, error)
	GetTransactions([]string) ([]byte, error)
	GetBallots() ([]byte, error)
}

type MessageBroker interface {
	Response(io.Writer, []byte) error
	Receive(common.NetworkMessage)
}
