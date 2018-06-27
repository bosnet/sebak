package sebakcommon

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"sync"

	"github.com/stellar/go/keypair"
)

var DefaultNodePort int = 12345

type Node interface {
	Address() string
	Keypair() *keypair.Full
	SetKeypair(*keypair.Full)
	Alias() string
	SetAlias(string)
	Endpoint() *Endpoint
	Equal(Node) bool
	DeepEqual(Node) bool
	GetValidators() map[string]*Validator
	AddValidators(validators ...*Validator) error
	HasValidators(string) bool
	RemoveValidators(validators ...*Validator) error
	Serialize() ([]byte, error)
}

type ValidatorFromJSON struct {
	Alias      string                `json:"alias"`
	Address    string                `json:"address"`
	Endpoint   *Endpoint             `json:"endpoint"`
	Validators map[string]*Validator `json:"validators"`
}

type Validator struct {
	sync.Mutex

	keypair *keypair.Full

	alias      string
	address    string
	endpoint   *Endpoint
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

func (v *Validator) Keypair() *keypair.Full {
	return v.keypair
}

func (v *Validator) SetKeypair(kp *keypair.Full) {
	v.address = kp.Address()
	v.keypair = kp
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

func (v *Validator) Endpoint() *Endpoint {
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
	var neighbors = make(map[string]struct{})
	for _, neighbor := range v.validators {
		neighbors[neighbor.Address()] = struct{}{}
	}
	return json.Marshal(map[string]interface{}{
		"address":    v.Address(),
		"alias":      v.Alias(),
		"endpoint":   v.Endpoint().String(),
		"validators": neighbors,
		//"validators": v.validators,
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
	v.validators = va.Validators

	return nil
}

func (v *Validator) Serialize() ([]byte, error) {
	return json.Marshal(v)
}

func MakeAlias(address string) string {
	l := len(address)
	return fmt.Sprintf("%s.%s", address[:4], address[l-8:l-4])
}

func NewValidator(address string, endpoint *Endpoint, alias string) (v *Validator, err error) {
	if len(alias) < 1 {
		alias = MakeAlias(address)
	}

	if _, err = keypair.Parse(address); err != nil {
		return
	}

	v = &Validator{
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

func ParseNodeEndpoint(endpoint string) (u *Endpoint, err error) {
	var parsed *url.URL
	parsed, err = url.Parse(endpoint)
	if err != nil {
		return
	}
	if len(parsed.Scheme) < 1 {
		err = errors.New("missing scheme")
		return
	}

	if len(parsed.Port()) < 1 && parsed.Scheme != "memory" {
		parsed.Host = fmt.Sprintf("%s:%d", parsed.Host, DefaultNodePort)
	}

	if parsed.Scheme != "memory" {
		var port string
		port = parsed.Port()

		var portInt int64
		if portInt, err = strconv.ParseInt(port, 10, 64); err != nil {
			return
		} else if portInt < 1 {
			err = errors.New("invalid port")
			return
		}

		if len(parsed.Host) < 1 || strings.HasPrefix(parsed.Host, "127.0.") {
			parsed.Host = fmt.Sprintf("localhost:%s", parsed.Port())
		}
	}

	parsed.Host = strings.ToLower(parsed.Host)

	u = (*Endpoint)(parsed)

	return
}
