package sebak

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/spikeekips/sebak/lib/network"
	"github.com/stellar/go/keypair"
)

var DefaultNodePort int = 12345

type Node interface {
	Address() string
	Alias() string
	Endpoint() *url.URL
}

type ValidatorNode interface {
	Address() string
	Alias() string
	Endpoint() *url.URL
	Transport() network.Transport
	SetTransport(network.Transport) error
	GetValidators() map[string]*Validator
	SetValidators(validators ...[]*Validator) error
}

// MainNode is the current node.
type MainNode struct {
	keypair   *keypair.Full
	endpoint  *url.URL
	transport network.Transport

	validators map[ /* Validator.Address() */ string]*Validator
}

func (n *MainNode) Keypair() *keypair.Full {
	return n.keypair
}

func (n *MainNode) Address() string {
	return n.keypair.Address()
}

func (n *MainNode) Alias() string {
	return "self"
}

func (n *MainNode) Endpoint() *url.URL {
	return n.endpoint
}

func (n *MainNode) SetTransport(t network.Transport) error {
	n.transport = t
	return nil
}

func (n *MainNode) CountValidators() int {
	return len(n.validators)
}

func (n *MainNode) GetValidators() map[string]*Validator {
	return n.validators
}

func (n *MainNode) SetValidators(validators ...*Validator) error {
	for _, v := range validators {
		n.validators[v.Address()] = v
	}

	return nil
}

func NewMainNode(kp *keypair.Full, endpoint *url.URL) (m *MainNode, err error) {
	m = &MainNode{
		keypair:    kp,
		endpoint:   endpoint,
		validators: map[string]*Validator{},
	}

	return
}

// Validator is the validator node for `MainNode`
type Validator struct {
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

func (v *Validator) Address() string {
	return v.address
}

func (v *Validator) Alias() string {
	return v.alias
}

func (v *Validator) Endpoint() *url.URL {
	return v.endpoint
}

func (v *Validator) SetTransport(t network.Transport) error {
	v.transport = t
	return nil
}

func (v *Validator) GetValidators() map[string]*Validator {
	return v.validators
}

func (v *Validator) SetValidators(validators ...*Validator) error {
	for _, va := range validators {
		v.validators[v.Address()] = va
	}

	return nil
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
