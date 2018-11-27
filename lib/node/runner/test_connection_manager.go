package runner

import (
	"sync"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/network"
	"boscoin.io/sebak/lib/node"
	"boscoin.io/sebak/lib/voting"
)

type TestConnectionManager struct {
	sync.RWMutex
	network.ConnectionManager

	messages []common.Message
	recv     chan struct{}
}

func NewTestConnectionManager(
	localNode *node.LocalNode,
	n network.Network,
	policy voting.ThresholdPolicy,
	r chan struct{},
) *TestConnectionManager {
	p := &TestConnectionManager{
		ConnectionManager: network.NewValidatorConnectionManager(localNode, n, policy),
	}
	p.messages = []common.Message{}
	p.recv = r

	return p
}

func (c *TestConnectionManager) Broadcast(message common.Message) {
	c.Lock()
	defer c.Unlock()
	c.messages = append(c.messages, message)
	if c.recv != nil {
		c.recv <- struct{}{}
	}
	return
}

func (c *TestConnectionManager) Messages() []common.Message {
	c.RLock()
	defer c.RUnlock()
	messages := make([]common.Message, len(c.messages))
	copy(messages, c.messages)
	return messages
}

func (c *TestConnectionManager) IsReady() bool {
	return true
}

type FixedSelector struct {
	address string
}

func (s FixedSelector) Select(_ uint64, _ uint64) string {
	return s.address
}

type OtherSelector struct {
	cm network.ConnectionManager
}

func (s OtherSelector) Select(_ uint64, _ uint64) string {
	for _, v := range s.cm.AllValidators() {
		if v != s.cm.GetNodeAddress() {
			return v
		}
	}
	panic("There is no the other validators")
}

type SelfThenOtherSelector struct {
	cm network.ConnectionManager
}

func (s SelfThenOtherSelector) Select(blockHeight uint64, round uint64) string {
	if blockHeight < 2 && round == 0 {
		return s.cm.GetNodeAddress()
	} else {
		for _, v := range s.cm.AllValidators() {
			if v != s.cm.GetNodeAddress() {
				return v
			}
		}
	}
	panic("There is no the other validators")
}
