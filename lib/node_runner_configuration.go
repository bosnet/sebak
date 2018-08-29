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

	TransactionsLimit uint64
}

func NewNodeRunnerConfiguration() *NodeRunnerConfiguration {
	p := NodeRunnerConfiguration{}

	p.TimeoutINIT = 2 * time.Second
	p.TimeoutSIGN = 2 * time.Second
	p.TimeoutACCEPT = 2 * time.Second
	p.TimeoutALLCONFIRM = 2 * time.Second
	p.TransactionsLimit = uint64(1000)

	return &p
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
