package sebaknode

import (
	"boscoin.io/sebak/lib/common"
)

type Node interface {
	Address() string
	Alias() string
	SetAlias(string)
	Endpoint() *sebakcommon.Endpoint
	Equal(Node) bool
	DeepEqual(Node) bool
	Serialize() ([]byte, error)
	State() NodeState
	SetBooting()
	SetSync()
	SetConsensus()
	SetTerminating()
}
