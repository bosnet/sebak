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

type Validator struct {
	sync.Mutex

	keypair *keypair.Full

	state      NodeState
	alias      string
	address    string
	endpoint   *sebakcommon.Endpoint
	validators map[ /* Node.Address() */ string]*Validator
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
	_, found := v.validators[address]
	return found
}

func (v *Validator) GetValidators() map[string]*Validator {
	return v.validators
}

func (v *Validator) AddValidators(validators ...*Validator) error {
	v.Lock()
	defer v.Unlock()

	for _, va := range validators {
		if v.Address() == va.Address() {
			continue
		}
		v.validators[va.Address()] = va
	}

	return nil
}

func (v *Validator) RemoveValidators(validators ...*Validator) error {
	v.Lock()
	defer v.Unlock()

	for _, va := range validators {
		if _, ok := v.validators[va.Address()]; !ok {
			continue
		}
		delete(v.validators, va.Address())
	}

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
	v.validators = va.Validators
	v.state = va.State

	return nil
}

func (v *Validator) Serialize() ([]byte, error) {
	return json.Marshal(v)
}

func MakeAlias(address string) string {
	l := len(address)
	return fmt.Sprintf("%s.%s", address[:4], address[l-8:l-4])
}

func NewValidator(address string, endpoint *sebakcommon.Endpoint, alias string) (v *Validator, err error) {
	if len(alias) < 1 {
		alias = MakeAlias(address)
	}

	if _, err = keypair.Parse(address); err != nil {
		return
	}

	v = &Validator{
		state:      NodeStateNONE,
		alias:      alias,
		address:    address,
		endpoint:   endpoint,
		validators: map[string]*Validator{},
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
