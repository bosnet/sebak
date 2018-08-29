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

func (n *NodeRunnerConfiguration) SetINIT(f float64) *NodeRunnerConfiguration {
	n.TimeoutINIT = Millisecond(f)
	return n
}

func (n *NodeRunnerConfiguration) SetSIGN(f float64) *NodeRunnerConfiguration {
	n.TimeoutSIGN = Millisecond(f)
	return n
}

func (n *NodeRunnerConfiguration) SetACCEPT(f float64) *NodeRunnerConfiguration {
	n.TimeoutACCEPT = Millisecond(f)
	return n
}

func (n *NodeRunnerConfiguration) SetALLCONFIRM(f float64) *NodeRunnerConfiguration {
	n.TimeoutALLCONFIRM = Millisecond(f)
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

func Millisecond(f float64) time.Duration {
	return time.Millisecond * time.Duration(int(f*1000))
}
