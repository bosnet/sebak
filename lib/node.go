package sebak

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"sync"

	"github.com/spikeekips/sebak/lib/network"
	"github.com/stellar/go/keypair"
)

var DefaultNodePort int = 12345

type Node interface {
	Address() string
	Keypair() *keypair.Full
	SetKeypair(*keypair.Full)
	Alias() string
	Endpoint() *url.URL
	Transport() network.Transport
	SetTransport(network.Transport) error
	GetValidators() map[string]*Validator
	AddValidators(validators ...*Validator) error
	RemoveValidators(validators ...*Validator) error
	Serialize() ([]byte, error)
}

type Validator struct {
	sync.Mutex

	keypair *keypair.Full

	alias     string
	address   string
	endpoint  *url.URL
	transport network.Transport

	validators map[ /* Node.Address() */ string]*Validator
}

func (v *Validator) String() string {
	o, _ := json.Marshal(map[string]string{
		"alias":    v.alias,
		"address":  v.address,
		"endpoint": v.endpoint.String(),
	})
	return string(o)
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

func (v *Validator) Endpoint() *url.URL {
	return v.endpoint
}

func (v *Validator) Transport() network.Transport {
	return v.transport
}

func (v *Validator) SetTransport(t network.Transport) error {
	v.transport = t
	return nil
}

func (v *Validator) GetValidators() map[string]*Validator {
	return v.validators
}

func (v *Validator) AddValidators(validators ...*Validator) error {
	v.Lock()
	defer v.Unlock()

	for _, va := range validators {
		v.validators[v.Address()] = va
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
		"address":    v.Address(),
		"alias":      v.Alias(),
		"endpoint":   v.Endpoint().String(),
		"validators": v.validators,
	})
}

func (v *Validator) Serialize() ([]byte, error) {
	return json.Marshal(v)
}

func NewValidator(address string, endpoint *url.URL, alias string) (v *Validator, err error) {
	if len(alias) < 1 {
		l := len(address)
		alias = fmt.Sprintf("%s.%s", address[:4], address[l-8:l-4])
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

func ParseNodeEndpoint(endpoint string) (u *url.URL, err error) {
	u, err = url.Parse(endpoint)
	if err != nil {
		return
	}
	if len(u.Scheme) < 1 {
		err = errors.New("missing scheme")
		return
	}

	if len(u.Port()) < 1 {
		u.Host = fmt.Sprintf("localhost:%d", DefaultNodePort)
	}

	var port string
	port = u.Port()

	var portInt int64
	if portInt, err = strconv.ParseInt(port, 10, 64); err != nil {
		return
	} else if portInt < 1 {
		err = errors.New("invalid port")
		return
	}

	if len(u.Host) < 1 || strings.HasPrefix(u.Host, "127.0.") {
		u.Host = fmt.Sprintf("localhost:%s", u.Port())
	}

	u.Host = strings.ToLower(u.Host)

	return
}
