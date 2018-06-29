package sebaknode

import (
	"encoding/json"
	"fmt"
	"sync"

	"boscoin.io/sebak/lib/common"

	"github.com/stellar/go/keypair"
)

type NodeFromJSON struct {
	Alias      string                `json:"alias"`
	Address    string                `json:"address"`
	Endpoint   *sebakcommon.Endpoint `json:"endpoint"`
	Validators map[string]*Validator `json:"Validators"`
	State      NodeState             `json:"state"`
}

type LocalNode struct {
	sync.Mutex

	keypair *keypair.Full

	state      NodeState
	alias      string
	address    string
	endpoint   *sebakcommon.Endpoint
	validators map[ /* Node.Address() */ string]*Validator
}

func (n *LocalNode) String() string {
	return n.Alias()
}

func (n *LocalNode) Equal(a Node) bool {
	if n.Address() == a.Address() {
		return true
	}

	return false
}

func (n *LocalNode) DeepEqual(a Node) bool {
	if !n.Equal(a) {
		return false
	}
	if n.Endpoint().String() != a.Endpoint().String() {
		return false
	}

	return true
}

func (n *LocalNode) State() NodeState {
	return n.state
}

func (n *LocalNode) SetBooting() {
	n.state = NodeStateBOOTING
}

func (n *LocalNode) SetCatchup() {
	n.state = NodeStateCATCHUP
}

func (n *LocalNode) SetConsensus() {
	n.state = NodeStateCONSENSUS
}

func (n *LocalNode) SetTerminating() {
	n.state = NodeStateTERMINATING
}

func (n *LocalNode) Address() string {
	return n.address
}

func (n *LocalNode) Keypair() *keypair.Full {
	return n.keypair
}

func (n *LocalNode) SetKeypair(kp *keypair.Full) {
	n.address = kp.Address()
	n.keypair = kp
}

func (n *LocalNode) Alias() string {
	return n.alias
}

func (n *LocalNode) SetAlias(s string) {
	n.alias = s
}

func (n *LocalNode) Endpoint() *sebakcommon.Endpoint {
	return n.endpoint
}

func (n *LocalNode) HasValidators(address string) bool {
	_, found := n.validators[address]
	return found
}

func (n *LocalNode) GetValidators() map[string]*Validator {
	return n.validators
}

func (n *LocalNode) AddValidators(validators ...*Validator) error {
	n.Lock()
	defer n.Unlock()

	for _, va := range validators {
		if n.Address() == va.Address() {
			continue
		}
		n.validators[va.Address()] = va
	}

	return nil
}

func (n *LocalNode) RemoveValidators(validators ...*Validator) error {
	n.Lock()
	defer n.Unlock()

	for _, va := range validators {
		if _, ok := n.validators[va.Address()]; !ok {
			continue
		}
		delete(n.validators, va.Address())
	}

	return nil
}

func (n *LocalNode) MarshalJSON() ([]byte, error) {
	var neighbors = make(map[string]struct{})
	for _, neighbor := range n.validators {
		neighbors[neighbor.Address()] = struct{}{}
	}
	return json.Marshal(map[string]interface{}{
		"address":    n.Address(),
		"alias":      n.Alias(),
		"endpoint":   n.Endpoint().String(),
		"state":      n.State().String(),
		"validators": neighbors,
	})
}

func (n *LocalNode) UnmarshalJSON(b []byte) error {
	var va NodeFromJSON
	if err := json.Unmarshal(b, &va); err != nil {
		return err
	}

	n.alias = va.Alias
	n.address = va.Address
	n.endpoint = va.Endpoint
	n.validators = va.Validators
	n.state = va.State

	return nil
}

func (n *LocalNode) Serialize() ([]byte, error) {
	return json.Marshal(n)
}

func (n *LocalNode) ConvertToValidator() *Validator {
	v, _ := NewValidator(n.Address(), n.Endpoint(), n.Alias())
	return v
}

func MakeAlias(address string) string {
	l := len(address)
	return fmt.Sprintf("%s.%s", address[:4], address[l-8:l-4])
}

func NewLocalNode(address string, endpoint *sebakcommon.Endpoint, alias string) (n *LocalNode, err error) {
	if len(alias) < 1 {
		alias = MakeAlias(address)
	}

	if _, err = keypair.Parse(address); err != nil {
		return
	}

	n = &LocalNode{
		state:      NodeStateNONE,
		alias:      alias,
		address:    address,
		endpoint:   endpoint,
		validators: map[string]*Validator{},
	}

	return
}

func NewLocalNodeFromString(b []byte) (*LocalNode, error) {
	var v LocalNode
	if err := json.Unmarshal(b, &v); err != nil {
		return nil, err
	}

	return &v, nil
}
