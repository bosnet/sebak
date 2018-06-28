package sebaknode

import (
	"encoding/json"
	"sync"

	"boscoin.io/sebak/lib/common"

	"github.com/stellar/go/keypair"
)

type LocalNodeFromJSON struct {
	Alias    string                `json:"alias"`
	Address  string                `json:"address"`
	Endpoint *sebakcommon.Endpoint `json:"endpoint"`
	State    NodeState             `json:"state"`
}

type Validator struct {
	sync.Mutex

	keypair *keypair.Full

	state    NodeState
	alias    string
	address  string
	endpoint *sebakcommon.Endpoint
}

func (v *Validator) String() string {
	return v.Alias()
}

func (v *Validator) Equal(a Node) bool {
	if v.Address() == a.Address() {
		return true
	}

	return false
}

func (v *Validator) DeepEqual(a Node) bool {
	if !v.Equal(a) {
		return false
	}
	if v.Endpoint().String() != a.Endpoint().String() {
		return false
	}

	return true
}

func (v *Validator) State() NodeState {
	return v.state
}

func (v *Validator) SetBooting() {
	v.state = NodeStateBOOTING
}

func (v *Validator) SetCatchup() {
	v.state = NodeStateCATCHUP
}

func (v *Validator) SetConsensus() {
	v.state = NodeStateCONSENSUS
}

func (v *Validator) SetTerminating() {
	v.state = NodeStateTERMINATING
}

func (v *Validator) Address() string {
	return v.address
}

func (v *Validator) Keypair() *keypair.Full {
	return v.keypair
}

func (v *Validator) SetKeypair(kp *keypair.Full) {
	v.address = kp.Address()
	v.keypair = kp
}

func (v *Validator) Alias() string {
	return v.alias
}

func (v *Validator) SetAlias(s string) {
	v.alias = s
}

func (v *Validator) Endpoint() *sebakcommon.Endpoint {
	return v.endpoint
}

func (v *Validator) HasValidators(address string) bool {
	return true
}

func (v *Validator) GetValidators() map[string]*Validator {
	return nil
}

func (v *Validator) AddValidators(validators ...*Validator) error {
	return nil
}

func (v *Validator) RemoveValidators(validators ...*Validator) error {
	return nil
}

func (v *Validator) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"address":  v.Address(),
		"alias":    v.Alias(),
		"endpoint": v.Endpoint().String(),
		"state":    v.State().String(),
		//"validators": v.validators,
	})
}

func (v *Validator) UnmarshalJSON(b []byte) error {
	var va NodeFromJSON
	if err := json.Unmarshal(b, &va); err != nil {
		return err
	}

	v.alias = va.Alias
	v.address = va.Address
	v.endpoint = va.Endpoint
	v.state = va.State

	return nil
}

func (v *Validator) Serialize() ([]byte, error) {
	return json.Marshal(v)
}

func NewValidatorFromURI(address string, endpoint *sebakcommon.Endpoint, alias string) (v *Validator, err error) {
	if len(alias) < 1 {
		alias = MakeAlias(address)
	}

	if _, err = keypair.Parse(address); err != nil {
		return
	}

	v = &Validator{
		state:    NodeStateNONE,
		alias:    alias,
		address:  address,
		endpoint: endpoint,
	}

	return
}

func NewValidatorFromString(b []byte) (*Validator, error) {
	var v Validator
	if err := json.Unmarshal(b, &v); err != nil {
		return nil, err
	}

	return &v, nil
}
