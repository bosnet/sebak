package sebaknode

import (
	"boscoin.io/sebak/lib/common"

	"github.com/stellar/go/keypair"
)

type Node interface {
	Address() string
	Keypair() *keypair.Full
	SetKeypair(*keypair.Full)
	Alias() string
	SetAlias(string)
	Endpoint() *sebakcommon.Endpoint
	Equal(Node) bool
	DeepEqual(Node) bool
	GetValidators() map[string]*Validator
	AddValidators(validators ...*Validator) error
	HasValidators(string) bool
	RemoveValidators(validators ...*Validator) error
	Serialize() ([]byte, error)
	State() NodeState
	SetBooting()
	SetCatchup()
	SetConsensus()
	SetTerminating()
}
