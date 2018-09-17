package consensus

import (
	"time"

	"boscoin.io/sebak/lib/ballot"
)

//
// ISAACConfiguration has timeout features and transaction limit.
// The ISAACConfiguration is included in ISAACStateManager and
// these timeout features are used in ISAAC consensus.
//
type ISAACConfiguration struct {
	TimeoutINIT       time.Duration
	TimeoutSIGN       time.Duration
	TimeoutACCEPT     time.Duration
	TimeoutALLCONFIRM time.Duration

	TransactionsLimit uint64
}

func NewISAACConfiguration() *ISAACConfiguration {
	p := ISAACConfiguration{}

	p.TimeoutINIT = 2 * time.Second
	p.TimeoutSIGN = 2 * time.Second
	p.TimeoutACCEPT = 2 * time.Second
	p.TimeoutALLCONFIRM = 2 * time.Second
	p.TransactionsLimit = uint64(1000)

	return &p
}

func (n *ISAACConfiguration) GetTimeout(ballotState ballot.State) time.Duration {
	switch ballotState {
	case ballot.StateINIT:
		return n.TimeoutINIT
	case ballot.StateSIGN:
		return n.TimeoutSIGN
	case ballot.StateACCEPT:
		return n.TimeoutACCEPT
	default:
		return 0
	}
}
