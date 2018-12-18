package network

import (
	"errors"
	"sync"
	"time"

	logging "github.com/inconshreveable/log15"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/metrics"
	"boscoin.io/sebak/lib/node"
	"boscoin.io/sebak/lib/voting"
)

type ValidatorConnectionManager struct {
	sync.RWMutex

	localNode *node.LocalNode
	network   Network
	policy    voting.ThresholdPolicy

	clients          map[ /* hash of node.Endpoint() */ string]NetworkClient
	connected        map[ /* node.Address() */ string]bool
	config           common.Config
	discoveryChannel chan DiscoveryMessage

	log logging.Logger
}

func NewValidatorConnectionManager(
	localNode *node.LocalNode,
	network Network,
	policy voting.ThresholdPolicy,
	config common.Config,
) ConnectionManager {
	if len(localNode.GetValidators()) == 0 {
		panic("empty validators")
	}
	cm := &ValidatorConnectionManager{
		localNode: localNode,
		network:   network,
		policy:    policy,
		config:    config,
		clients:   map[string]NetworkClient{},
		connected: map[string]bool{},
		log:       log.New(logging.Ctx{"node": localNode.Alias()}),
	}
	cm.connected[localNode.Address()] = true
	cm.discoveryChannel = make(chan DiscoveryMessage, 100)

	return cm
}

func (c *ValidatorConnectionManager) GetConnection(address string) (client NetworkClient) {
	var validator *node.Validator
	if validator = c.localNode.Validator(address); validator == nil {
		return
	}

	return c.GetConnectionByEndpoint(validator.Endpoint())
}

func (c *ValidatorConnectionManager) GetConnectionByEndpoint(endpoint *common.Endpoint) (client NetworkClient) {
	c.Lock()
	defer c.Unlock()

	hash := common.MustMakeObjectHashString(endpoint)

	var ok bool
	client, ok = c.clients[hash]
	if ok {
		return
	}

	client = c.network.GetClient(endpoint)
	if client != nil {
		c.clients[hash] = client
	}

	return
}

func (c *ValidatorConnectionManager) Start() {
	if !c.config.WatcherMode {
		c.log.Debug("starting discovery of validators", "validators", c.localNode.GetValidators())

		// wait until enough discovered; over threshold
		c.startDiscovery()
	}

	c.log.Debug("starting to connect to validators", "validators", c.localNode.GetValidators())
	for _, v := range c.localNode.GetValidators() {
		if v.Address() == c.localNode.Address() {
			continue
		}
		go c.connectingValidator(v)
	}
	go c.watchForMetrics()
}

// setConnected returns `true` when the validator is newly connected or
// disconnected at first
func (c *ValidatorConnectionManager) setConnected(v *node.Validator, connected bool) bool {
	c.Lock()
	defer c.Unlock()

	old, found := c.connected[v.Address()]
	c.connected[v.Address()] = connected

	c.policy.SetConnected(c.countConnectedUnlocked())
	return !found || old != connected
}

func (c *ValidatorConnectionManager) AllConnected() []string {
	c.RLock()
	defer c.RUnlock()
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
//   A list of all validators
func (c *ValidatorConnectionManager) AllValidators() []string {
	var validators []string
	for address := range c.localNode.GetValidators() {
		validators = append(validators, address)
	}
	return validators
}

//
// Returns:
//   the number of validators which are currently connected
//
func (c *ValidatorConnectionManager) CountConnected() int {
	c.RLock()
	defer c.RUnlock()
	return c.countConnectedUnlocked()
}

func (c *ValidatorConnectionManager) countConnectedUnlocked() int {
	var count int
	for _, isConnected := range c.connected {
		if isConnected {
			count += 1
		}
	}
	return count
}

func (c *ValidatorConnectionManager) connectingValidator(v *node.Validator) {
	ticker := time.NewTicker(time.Second * 1)
	for _ = range ticker.C {
		if v.Endpoint() == nil {
			continue
		}

		err := c.connectValidator(v)

		if c.setConnected(v, err == nil) {
			if err == nil {
				c.log.Debug("validator is connected", "validator", v.Address())
			} else {
				c.log.Debug("validator is disconnected", "validator", v.Address(), "error", err)
			}
		}
	}

	return
}

func (c *ValidatorConnectionManager) connectValidator(v *node.Validator) (err error) {
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

func (c *ValidatorConnectionManager) Broadcast(message common.Message) {
	c.RLock()
	defer c.RUnlock()

	for addr, connected := range c.connected {
		var validator *node.Validator
		if validator = c.localNode.Validator(addr); validator == nil {
			continue
		}
		if validator.Address() == c.localNode.Address() {
			continue
		}
		if validator.Endpoint() == nil {
			continue
		}

		if !connected {
			continue
		}

		go func(v *node.Validator) {
			client := c.GetConnection(v.Address())

			var err error
			var response []byte
			if message.GetType() == common.BallotMessage {
				response, err = client.SendBallot(message)
			} else if message.GetType() == common.TransactionMessage {
				response, err = client.SendMessage(message)
			} else if message.GetType() == common.DiscoveryMessage {
				response, err = client.SendDiscovery(message)
			} else {
				panic("invalid message")
			}

			if err != nil {
				c.log.Error(
					"failed to broadcast",
					"error", err,
					"validator", v.Address(),
					"type", message.GetType(),
					"message", message.GetHash(),
					"response", string(response),
				)
			}
		}(validator)
	}

	return
}

func (c *ValidatorConnectionManager) watchForMetrics() {
	numValidators := len(c.localNode.GetValidators())
	metrics.Consensus.SetValidators(numValidators)

	ticker := time.NewTicker(time.Second * 60)
	for _ = range ticker.C {
		numConnected := c.CountConnected()
		metrics.Consensus.SetMissingValidators(numValidators - numConnected)
	}
	//TODO: stop this goroutine.
}

func (c *ValidatorConnectionManager) IsReady() bool {
	return len(c.AllConnected()) >= c.policy.Threshold()
}

// startDiscovery will try to discover the validators; it will block until
// enough validators is discovered, and then it will keep discovering the left
// validators.
func (c *ValidatorConnectionManager) startDiscovery() {
	go func() {
		for {
			select {
			case dm := <-c.discoveryChannel:
				c.discovery(dm)
			}
		}
	}()

	broadcastTicker := time.NewTicker(time.Millisecond * 500)
	go func() {
		for _ = range broadcastTicker.C {
			c.broadcastDiscovery()
		}
	}()

	ticker := time.NewTicker(time.Millisecond * 300)
	for _ = range ticker.C {
		if len(c.discovered()) >= c.policy.Threshold() {
			ticker.Stop()
			break
		}
	}
	broadcastTicker.Stop()

	go func() {
		for {
			if len(c.discovered()) == len(c.localNode.GetValidators()) {
				break
			}

			c.broadcastDiscovery()
			time.Sleep(time.Millisecond * 500)
		}
	}()
}

func (c *ValidatorConnectionManager) Discovery(dm DiscoveryMessage) error {
	c.discoveryChannel <- dm

	return nil
}

func (c *ValidatorConnectionManager) discovered() (vs []*node.Validator) {
	for _, v := range c.localNode.GetValidators() {
		if v.Endpoint() == nil {
			continue
		}

		vs = append(vs, v)
	}

	return
}

func (c *ValidatorConnectionManager) discovery(dm DiscoveryMessage) (err error) {
	clog := c.log.New(logging.Ctx{
		"dm":         dm.GetHash(),
		"from":       dm.Source(),
		"validators": dm.B.Validators,
	})

	clog.Debug("got DiscoveryMessage")

	current := c.discovered()

	discovered := dm.FilterUndiscovered(c.localNode.GetValidators())
	if len(discovered) > 0 {
		c.log.Debug(
			"new validators found",
			"from", dm.B.Address,
			"received", dm.B.Validators,
			"new", discovered,
			"previous", current,
			"after", c.discovered(),
		)

		c.setDiscovered(discovered...)
		c.broadcastDiscovery()

		return
	}

	clog.Debug(
		"new discovery not found",
		"from", dm.B.Address,
		"received", dm.B.Validators,
		"current", current,
	)

	// if received DiscoveryMessage does not have discovered validators,
	// broadcast to him.
	var shouldBroadcast bool
	for _, lv := range c.localNode.GetValidators() {
		var rv *node.Validator
		for _, v := range dm.B.Validators {
			if v.Address() == lv.Address() {
				rv = v
				break
			}
		}
		if rv != nil {
			continue
		}

		if lv.Endpoint() != nil {
			shouldBroadcast = true
			break
		}
	}

	if shouldBroadcast {
		c.broadcastDiscovery(dm.B.Endpoint)
	}

	return
}

func (c *ValidatorConnectionManager) setDiscovered(vs ...*node.Validator) {
	for _, v := range vs {
		lv := c.localNode.Validator(v.Address())
		if lv == nil {
			continue
		}

		lv.SetEndpoint(v.Endpoint())
	}
}

func (c *ValidatorConnectionManager) broadcastDiscovery(endpoints ...*common.Endpoint) {
	var err error
	var dm DiscoveryMessage
	if dm, err = NewDiscoveryMessage(c.localNode, c.discovered()...); err != nil {
		c.log.Error("failed to make DiscoveryMessage", "discovered", c.discovered(), "error", err)
		return
	}
	dm.Sign(c.localNode.Keypair(), c.config.NetworkID)

	// if the argument, endpoints is empty, broadcast to all.
	if len(endpoints) < 1 {
		endpoints = append(endpoints, c.config.DiscoveryEndpoints...)

		for addr, connected := range c.connected {
			var validator *node.Validator
			if validator = c.localNode.Validator(addr); validator == nil {
				continue
			}
			if validator.Address() == c.localNode.Address() {
				continue
			}
			if validator.Endpoint() == nil {
				continue
			}

			if !connected {
				continue
			}

			endpoints = append(endpoints, validator.Endpoint())
		}
	}

	for _, endpoint := range endpoints {
		go func(v *common.Endpoint) {
			client := c.GetConnectionByEndpoint(v)

			if response, err := client.SendDiscovery(dm); err != nil {
				c.log.Error(
					"failed to broadcast DiscoveryMessage",
					"error", err,
					"endpoint", v,
					"message", dm.GetHash(),
					"response", string(response),
				)
			}
		}(endpoint)
	}

	return
}
