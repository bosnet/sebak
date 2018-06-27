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

type NodeFromJSON struct {
	Alias      string           `json:"alias"`
	Address    string           `json:"address"`
	Endpoint   *Endpoint        `json:"endpoint"`
	Validators map[string]*Node `json:"Validators"`
}

type Node struct {
	sync.Mutex

	keypair *keypair.Full

	state      NodeState
	alias      string
	address    string
	endpoint   *Endpoint
	validators map[ /* Node.Address() */ string]*Node
}

func (n *Node) String() string {
	return n.Alias()
}

func (n *Node) Equal(a Node) bool {
	if n.Address() == a.Address() {
		return true
	}

	return false
}

func (n *Node) DeepEqual(a Node) bool {
	if !n.Equal(a) {
		return false
	}
	if n.Endpoint().String() != a.Endpoint().String() {
		return false
	}

	return true
}

func (n *Node) State() NodeState {
	return n.state
}

func (n *Node) Address() string {
	return n.address
}

func (n *Node) Keypair() *keypair.Full {
	return n.keypair
}

func (n *Node) SetKeypair(kp *keypair.Full) {
	n.address = kp.Address()
	n.keypair = kp
}

func (n *Node) Alias() string {
	return n.alias
}

func (n *Node) SetAlias(s string) {
	n.alias = s
}

func (n *Node) Endpoint() *Endpoint {
	return n.endpoint
}

func (n *Node) HasValidators(address string) bool {
	_, found := n.validators[address]
	return found
}

func (n *Node) GetValidators() map[string]*Node {
	return n.validators
}

func (n *Node) AddValidators(validators ...*Node) error {
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

func (n *Node) RemoveValidators(validators ...*Node) error {
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

func (v *Node) MarshalJSON() ([]byte, error) {
	var neighbors = make(map[string]struct{})
	for _, neighbor := range v.validators {
		neighbors[neighbor.Address()] = struct{}{}
	}
	return json.Marshal(map[string]interface{}{
		"address":    v.Address(),
		"alias":      v.Alias(),
		"endpoint":   v.Endpoint().String(),
		"validators": neighbors,
	})
}

func (n *Node) UnmarshalJSON(b []byte) error {
	var va NodeFromJSON
	if err := json.Unmarshal(b, &va); err != nil {
		return err
	}

	n.alias = va.Alias
	n.address = va.Address
	n.endpoint = va.Endpoint
	n.validators = va.Validators

	return nil
}

func (n *Node) Serialize() ([]byte, error) {
	return json.Marshal(n)
}

func MakeAlias(address string) string {
	l := len(address)
	return fmt.Sprintf("%s.%s", address[:4], address[l-8:l-4])
}

func NewNode(address string, endpoint *Endpoint, alias string) (n *Node, err error) {
	if len(alias) < 1 {
		alias = MakeAlias(address)
	}

	if _, err = keypair.Parse(address); err != nil {
		return
	}

	n = &Node{
		state:      NodeStateBOOTING,
		alias:      alias,
		address:    address,
		endpoint:   endpoint,
		validators: map[string]*Node{},
	}

	return
}

func NewNodeFromString(b []byte) (*Node, error) {
	var v Node
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

func NewValidatorFromURI(v string) (validator *Node, err error) {
	var parsed *url.URL
	if parsed, err = url.Parse(v); err != nil {
		return
	}

	var endpoint *Endpoint
	endpoint, err = ParseNodeEndpoint(
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

	if validator, err = NewNode(address, endpoint, alias); err != nil {
		return
	}

	return
}
