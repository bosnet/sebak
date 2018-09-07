//
// IsaacConfiguration has timeout features and transaction limit.
// The IsaacConfiguration is included in IsaacStateManager and
// these timeout features are used in ISAAC consensus.
//
package sebak

import (
	"time"

	"boscoin.io/sebak/lib/common"
)

type IsaacConfiguration struct {
	TimeoutINIT       time.Duration
	TimeoutSIGN       time.Duration
	TimeoutACCEPT     time.Duration
	TimeoutALLCONFIRM time.Duration

	TransactionsLimit uint64
}

func NewIsaacConfiguration() *IsaacConfiguration {
	p := IsaacConfiguration{}

	p.TimeoutINIT = 2 * time.Second
	p.TimeoutSIGN = 2 * time.Second
	p.TimeoutACCEPT = 2 * time.Second
	p.TimeoutALLCONFIRM = 2 * time.Second
	p.TransactionsLimit = uint64(1000)

	return &p
}

func (n *IsaacConfiguration) GetTimeout(ballotState common.BallotState) time.Duration {
	switch ballotState {
	case common.BallotStateINIT:
		return n.TimeoutINIT
	case common.BallotStateSIGN:
		return n.TimeoutSIGN
	case common.BallotStateACCEPT:
		return n.TimeoutACCEPT
	default:
		return 0
	}
}
