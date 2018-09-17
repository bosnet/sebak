package node

import (
	"boscoin.io/sebak/lib/common"
)

type Node interface {
	Address() string
	Alias() string
	Endpoint() *common.Endpoint
	Equal(Node) bool
	Serialize() ([]byte, error)
	State() State
	SetBooting()
	SetSync()
	SetConsensus()
	SetTerminating()
}
