package runner

import (
	"sync"

	"boscoin.io/sebak/lib/ballot"
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/network"
	"boscoin.io/sebak/lib/node"
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
	policy ballot.VotingThresholdPolicy,
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

type SelfSelector struct {
	cm network.ConnectionManager
}

func (s SelfSelector) Select(_ uint64, _ uint64) string {
	return s.cm.GetNodeAddress()
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

func (s SelfThenOtherSelector) Select(blockHeight uint64, roundNumber uint64) string {
	if blockHeight < 2 && roundNumber == 0 {
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
