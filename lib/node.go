package sebak

import (
	"fmt"

	"github.com/stellar/go/keypair"
)

type Node interface {
	Address() string
	Alias() string
	Endpoint() string
}

type ValidatorNode interface {
	Address() string
	Alias() string
	Endpoint() string
	Transport() Transport
	SetTransport(Transport) error
}

// MainNode is the current node.
type MainNode struct {
	keypair   *keypair.Full
	transport Transport

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

func (n *MainNode) Endpoint() string {
	return n.transport.Endpoint()
}

func (n *MainNode) SetTransport(t Transport) error {
	n.transport = t
	return nil
}

func NewMainNode(kp *keypair.Full, transport Transport) *MainNode {
	return &MainNode{
		keypair:   kp,
		transport: transport,
	}
}

// Validator is the validator node for `MainNode`
type Validator struct {
	address   string
	endpoint  string
	transport Transport

	validators map[ /* Node.Address() */ string]*Validator
}

func (v *Validator) Address() string {
	return v.address
}

func (v *Validator) Alias() string {
	l := len(v.address)
	return fmt.Sprintf("%s...%s", v.address[:4], v.address[l-8:l-4])
}

func (v *Validator) Endpoint() string {
	endpoint := v.endpoint
	if v.transport == nil {
		endpoint = v.transport.Endpoint()
	}

	return endpoint
}

func (v *Validator) SetTransport(t Transport) error {
	v.transport = t
	return nil
}

func NewValidator(address, endpoint string) *Validator {
	return &Validator{
		address:  address,
		endpoint: endpoint,
	}
}
