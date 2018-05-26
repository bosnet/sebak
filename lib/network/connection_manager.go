package sebaknetwork

import (
	"errors"
	"net"
	"net/http"
	"sync"
	"time"

	logging "github.com/inconshreveable/log15"
	"github.com/spikeekips/sebak/lib/common"
)

type ConnectionManager struct {
	sync.Mutex

	currentNode sebakcommon.Node
	network     Network
	policy      sebakcommon.VotingThresholdPolicy

	validators map[ /* nodd.Address() */ string]*sebakcommon.Validator
	clients    map[ /* nodd.Address() */ string]NetworkClient
	connected  map[ /* nodd.Address() */ string]bool

	log logging.Logger
}

func NewConnectionManager(
	currentNode sebakcommon.Node,
	network Network,
	policy sebakcommon.VotingThresholdPolicy,
	validators map[string]*sebakcommon.Validator,
) *ConnectionManager {
	return &ConnectionManager{
		currentNode: currentNode,

		network:    network,
		policy:     policy,
		validators: validators,

		clients:   map[string]NetworkClient{},
		connected: map[string]bool{},
		log:       log.New(logging.Ctx{"node": currentNode.Alias()}),
	}
}

func (c *ConnectionManager) GetConnection(address string) (client NetworkClient) {
	c.Lock()
	defer c.Unlock()

	var ok bool
	client, ok = c.clients[address]
	if ok {
		return
	}

	var validator *sebakcommon.Validator
	if validator, ok = c.validators[address]; !ok {
		return
	}

	client = c.network.GetClient(validator.Endpoint())
	if client != nil {
		c.clients[address] = client
	}

	return
}

func (c *ConnectionManager) Start() {
	go c.connectValidators()
}

// setConnected returns `true` when the validator is newly connected or
// disconnected at first
func (c *ConnectionManager) setConnected(v *sebakcommon.Validator, connected bool) bool {
	c.Lock()
	defer c.Unlock()
	defer func() {
		c.policy.SetConnected(c.CountConnected())
	}()

	_, ok := c.connected[v.Address()]
	if !connected {
		delete(c.connected, v.Address())
		return ok
	}
	if ok {
		return false
	}

	c.connected[v.Address()] = true

	return true
}

func (c *ConnectionManager) IsConnected(v *sebakcommon.Validator) bool {
	_, ok := c.connected[v.Address()]

	return ok
}

func (c *ConnectionManager) AllConnected() []*sebakcommon.Validator {
	var connected []*sebakcommon.Validator
	for address := range c.connected {
		connected = append(connected, c.validators[address])
	}

	return connected
}

func (c *ConnectionManager) CountConnected() int {
	return len(c.connected)
}

func (c *ConnectionManager) connectValidators() {
	c.log.Debug("> starting to connect to validators", "validators", c.validators)
	for _, v := range c.validators {
		go c.connectingValidator(v)
	}

	select {}
}

func (c *ConnectionManager) connectingValidator(v *sebakcommon.Validator) {
	ticker := time.NewTicker(time.Second * 1)
	for _ = range ticker.C {
		err := c.connectValidator(v)
		if err != nil {
			c.log.Error("failed to connect", "validator", v, "error", err)
			continue
		}

		if c.setConnected(v, err == nil) {
			if err == nil {
				c.log.Debug("validator is connected", "validator", v)
			} else {
				c.log.Debug("validator is disconnected", "validator", v)
			}
		}
	}

	return
}

func (c *ConnectionManager) connectValidator(v *sebakcommon.Validator) (err error) {
	client := c.GetConnection(v.Address())

	var b []byte
	b, err = client.Connect(c.currentNode)
	if err != nil {
		return
	}

	/* load and check validator info; addresses are same?
	 */
	var validator *sebakcommon.Validator
	validator, err = sebakcommon.NewValidatorFromString(b)
	if err != nil {
		return
	}
	if v.Address() != validator.Address() {
		err = errors.New("address is mismatch")
		return
	}

	return
}

func (c *ConnectionManager) ConnectionWatcher(t Network, conn net.Conn, state http.ConnState) {
	return
}

func (c *ConnectionManager) Broadcast(message sebakcommon.Message) {
	for _, validator := range c.AllConnected() {
		go func(v *sebakcommon.Validator) {
			client := c.GetConnection(v.Address())
			if err := client.SendBallot(message); err != nil {
				c.log.Error("failed to SendBallot", "error", err, "validator", v)
			}
		}(validator)
	}
}
