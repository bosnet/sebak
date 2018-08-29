//
// NodeRunnerConfiguration has timeout features and transaction limit.
// The NodeRunnerConfiguration is included in NodeRunnerStateManager and
// these timeout features are used in ISAAC consensus.
//
package sebak

import (
	"time"

	"boscoin.io/sebak/lib/common"
)

type NodeRunnerConfiguration struct {
	TimeoutINIT       time.Duration
	TimeoutSIGN       time.Duration
	TimeoutACCEPT     time.Duration
	TimeoutALLCONFIRM time.Duration

	TransactionsLimit int
}

func NewNodeRunnerConfiguration() *NodeRunnerConfiguration {
	p := NodeRunnerConfiguration{}
	p.SetINIT(2).SetSIGN(2).SetACCEPT(2).SetALLCONFIRM(2).SetTxLimit(1000)
	return &p
}

func (n *NodeRunnerConfiguration) SetINIT(t time.Duration) *NodeRunnerConfiguration {
	n.TimeoutINIT = t
	return n
}

func (n *NodeRunnerConfiguration) SetSIGN(t time.Duration) *NodeRunnerConfiguration {
	n.TimeoutSIGN = t
	return n
}

func (n *NodeRunnerConfiguration) SetACCEPT(t time.Duration) *NodeRunnerConfiguration {
	n.TimeoutACCEPT = t
	return n
}

func (n *NodeRunnerConfiguration) SetALLCONFIRM(t time.Duration) *NodeRunnerConfiguration {
	n.TimeoutALLCONFIRM = t
	return n
}

func (n *NodeRunnerConfiguration) GetTimeout(ballotState sebakcommon.BallotState) time.Duration {
	switch ballotState {
	case sebakcommon.BallotStateINIT:
		return n.TimeoutINIT
	case sebakcommon.BallotStateSIGN:
		return n.TimeoutSIGN
	case sebakcommon.BallotStateACCEPT:
		return n.TimeoutACCEPT
	default:
		return 0
	}
}

func (n *NodeRunnerConfiguration) SetTxLimit(i int) *NodeRunnerConfiguration {
	n.TransactionsLimit = i
	return n
}
