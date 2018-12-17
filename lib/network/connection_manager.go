package network

import (
	"boscoin.io/sebak/lib/common"
)

type ConnectionManager interface {
	GetConnection(string) NetworkClient
	Broadcast(common.Message)
	Start()
	AllConnected() []string
	AllValidators() []string
	CountConnected() int
	IsReady() bool
	Discovery(DiscoveryMessage) error
}
