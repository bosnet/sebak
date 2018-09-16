//
// Defines the `Validator` type of Node, which is a remote node
//
// A `Validator` is a remote node as seen by the other type of node (`LocalNode`).
// It provides any information which is node-specific and relevant to us / consensus.
//
package node

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"sync"

	"boscoin.io/sebak/lib/common"

	"github.com/stellar/go/keypair"
)

type ValidatorFromJSON struct {
	Alias    string           `json:"alias"`
	Address  string           `json:"address"`
	Endpoint *common.Endpoint `json:"endpoint"`
	State    State            `json:"state"`
}

type Validator struct {
	sync.Mutex

	state    State
	alias    string
	address  string
	endpoint *common.Endpoint
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

func (v *Validator) State() State {
	return v.state
}

func (v *Validator) SetBooting() {
	v.state = StateBOOTING
}

func (v *Validator) SetSync() {
	v.state = StateSYNC
}

func (v *Validator) SetConsensus() {
	v.state = StateCONSENSUS
}

func (v *Validator) SetTerminating() {
	v.state = StateTERMINATING
}

func (v *Validator) Address() string {
	return v.address
}

func (v *Validator) Alias() string {
	return v.alias
}

func (v *Validator) SetAlias(s string) {
	v.alias = s
}

func (v *Validator) Endpoint() *common.Endpoint {
	return v.endpoint
}

func (v *Validator) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"address":  v.Address(),
		"alias":    v.Alias(),
		"endpoint": v.Endpoint().String(),
		"state":    v.State().String(),
	})
}

func (v *Validator) UnmarshalJSON(b []byte) error {
	var va ValidatorFromJSON
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

func NewValidator(address string, endpoint *common.Endpoint, alias string) (v *Validator, err error) {
	if len(alias) < 1 {
		alias = MakeAlias(address)
	}

	if _, err = keypair.Parse(address); err != nil {
		return
	}

	v = &Validator{
		state:    StateNONE,
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

func NewValidatorFromURI(v string) (validator *Validator, err error) {
	var parsed *url.URL
	if parsed, err = url.Parse(v); err != nil {
		return
	}

	var endpoint *common.Endpoint
	endpoint, err = common.ParseEndpoint(
		fmt.Sprintf("%s://%s%s", parsed.Scheme, parsed.Host, parsed.Path),
	)
	if err != nil {
		return
	}

	queries := parsed.Query()

	var address, alias string
	if addressStrings, ok := queries["address"]; !ok || len(addressStrings) < 1 {
		err = errors.New("`address` is missing")
		return
	} else {
		var parsedKP keypair.KP
		if parsedKP, err = keypair.Parse(addressStrings[0]); err != nil {
			return
		}
		address = parsedKP.Address()
	}

	if aliasStrings, ok := queries["alias"]; ok && len(aliasStrings) > 0 {
		alias = aliasStrings[0]
	}

	if validator, err = NewValidator(address, endpoint, alias); err != nil {
		return
	}

	return
}
