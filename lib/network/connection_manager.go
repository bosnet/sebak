package sebaknetwork

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

	localNode *sebaknode.LocalNode
	network   Network
	policy    sebakcommon.VotingThresholdPolicy

	validators map[ /* nodd.Address() */ string]*sebaknode.Validator
	clients    map[ /* nodd.Address() */ string]NetworkClient
	connected  map[ /* nodd.Address() */ string]bool

	log logging.Logger
}

func NewConnectionManager(
	localNode *sebaknode.LocalNode,
	network Network,
	policy sebakcommon.VotingThresholdPolicy,
	validators map[string]*sebaknode.Validator,
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

	var validator *sebaknode.Validator
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
func (c *ConnectionManager) setConnected(v *sebaknode.Validator, connected bool) bool {
	c.Lock()
	defer c.Unlock()
	defer func() {
		c.policy.SetConnected(c.CountConnected())
	}()

	old, found := c.connected[v.Address()]
	c.connected[v.Address()] = connected

	return !found || old != connected
}

func (c *ConnectionManager) IsConnected(v *sebaknode.Validator) bool {
	connected, ok := c.connected[v.Address()]
	if !ok {
		return false
	}

	return connected
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

func (c *ConnectionManager) RoundCandidates() []string {
	return append(c.AllConnected(), c.localNode.Address())
}

func (c *ConnectionManager) CountConnected() int {
	return len(c.AllConnected())
}

func (c *ConnectionManager) connectValidators() {
	c.log.Debug("> starting to connect to validators", "validators", c.validators)
	for _, v := range c.validators {
		go c.connectingValidator(v)
	}

	select {}
}

func (c *ConnectionManager) connectingValidator(v *sebaknode.Validator) {
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

func (c *ConnectionManager) connectValidator(v *sebaknode.Validator) (err error) {
	client := c.GetConnection(v.Address())

	var b []byte
	b, err = client.Connect(c.localNode)
	if err != nil {
		return
	}

	// load and check validator info; addresses are same?
	var validator *sebaknode.Validator
	validator, err = sebaknode.NewValidatorFromString(b)
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
	for _, address := range c.AllConnected() {
		go func(v string) {
			client := c.GetConnection(v)

			var err error
			if message.GetType() == BallotMessage {
				_, err = client.SendBallot(message)
			} else if message.GetType() == RoundBallotMessage {
				_, err = client.SendRoundBallot(message)
			} else {
				panic("invalid message")
			}

			if err != nil {
				c.log.Error("failed to SendBallot", "error", err, "validator", v)
			}
		}(address)
	}
}
