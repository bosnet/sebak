package network

import (
	"net"
	"net/http"

	"boscoin.io/sebak/lib/common"
)

type ConnectionManager interface {
	GetNodeAddress() string
	ConnectionWatcher(Network, net.Conn, http.ConnState)
	Broadcast(common.Message)
	Start()
	AllConnected() []string
	AllValidators() []string
	CountConnected() int
}
