package network

import (
	"io"
	"net"
	"net/http"

	"github.com/gorilla/mux"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/node"
)

type Network interface {
	Endpoint() *common.Endpoint
	GetClient(endpoint *common.Endpoint) NetworkClient
	AddWatcher(func(Network, net.Conn, http.ConnState))
	AddHandler(string, http.HandlerFunc) *mux.Route

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
	SendMessage(common.Serializable) ([]byte, error)
	SendBallot(common.Serializable) ([]byte, error)
	GetTransactions([]string) ([]byte, error)
}

type MessageBroker interface {
	Response(io.Writer, []byte) error
	Receive(common.NetworkMessage)
}
