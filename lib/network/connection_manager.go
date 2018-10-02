package network

import (
	"net"
	"net/http"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/node"
)

type ConnectionManager interface {
	GetNodeAddress() string
	GetConnection(string) NetworkClient
	ConnectionWatcher(Network, net.Conn, http.ConnState)
	Broadcast(common.Message)
	Start()
	AllConnected() []string
	AllValidators() []string
	CountConnected() int
	GetNode(address string) node.Node
}
