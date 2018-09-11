package network

import (
	"errors"
	"net"
	"net/http"
	"sync"
	"time"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/node"
	logging "github.com/inconshreveable/log15"
)

type ConnectionManager struct {
	sync.Mutex

	localNode *node.LocalNode
	network   Network
	policy    common.VotingThresholdPolicy

	validators map[ /* nodd.Address() */ string]*node.Validator
	clients    map[ /* nodd.Address() */ string]NetworkClient
	connected  map[ /* nodd.Address() */ string]bool

	log logging.Logger
}

func NewConnectionManager(
	localNode *node.LocalNode,
	network Network,
	policy common.VotingThresholdPolicy,
	validators map[string]*node.Validator,
) *ConnectionManager {
	return &ConnectionManager{
		localNode: localNode,

		network:    network,
		policy:     policy,
		validators: validators,

		clients:   map[string]NetworkClient{},
		connected: map[string]bool{},
		log:       log.New(logging.Ctx{"node": localNode.Alias()}),
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

	var validator *node.Validator
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
	c.log.Debug("starting to connect to validators", "validators", c.validators)
	for _, v := range c.validators {
		go c.connectingValidator(v)
	}
}

// setConnected returns `true` when the validator is newly connected or
// disconnected at first
func (c *ConnectionManager) setConnected(v *node.Validator, connected bool) bool {
	c.Lock()
	defer c.Unlock()
	defer func() {
		c.policy.SetConnected(c.CountConnected())
	}()

	old, found := c.connected[v.Address()]
	c.connected[v.Address()] = connected

	return !found || old != connected
}

func (c *ConnectionManager) AllConnected() []string {
	var connected []string
	for address, isConnected := range c.connected {
		if !isConnected {
			continue
		}
		connected = append(connected, address)
	}

	return connected
}

// Returns:
//   A list of all validators, including self
func (c *ConnectionManager) AllValidators() []string {
	var validators []string
	for address := range c.validators {
		validators = append(validators, address)
	}
	return append(validators, c.localNode.Address())
}

//
// Returns:
//   the number of validators which are currently connected
//
func (c *ConnectionManager) CountConnected() int {
	var count int
	for _, isConnected := range c.connected {
		if isConnected {
			count += 1
		}
	}
	return count
}

func (c *ConnectionManager) connectingValidator(v *node.Validator) {
	ticker := time.NewTicker(time.Second * 1)
	for _ = range ticker.C {
		err := c.connectValidator(v)

		if c.setConnected(v, err == nil) {
			if err == nil {
				c.log.Debug("validator is connected", "validator", v)
			} else {
				c.log.Debug("validator is disconnected", "validator", v, "error", err)
			}
		}
	}

	return
}

func (c *ConnectionManager) connectValidator(v *node.Validator) (err error) {
	client := c.GetConnection(v.Address())

	var b []byte
	b, err = client.Connect(c.localNode)
	if err != nil {
		return
	}

	// load and check validator info; addresses are same?
	var validator *node.Validator
	validator, err = node.NewValidatorFromString(b)
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

func (c *ConnectionManager) Broadcast(message common.Message) {
	for address, connected := range c.connected {
		if connected {
			go func(v string) {
				client := c.GetConnection(v)

				var err error
				if message.GetType() == common.BallotMessage {
					_, err = client.SendBallot(message)
				} else if message.GetType() == string(common.TransactionMessage) {
					_, err = client.SendMessage(message)
				} else {
					panic("invalid message")
				}

				if err != nil {
					c.log.Error("failed to SendBallot", "error", err, "validator", v)
				}
			}(address)
		}
	}
}
