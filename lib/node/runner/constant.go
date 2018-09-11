package runner

import (
	"time"
)

var (
	// TimeoutProposeNewBallot works when the consensus process is finished for
	// a proposed `Ballot`, proposer will wait for `TimeoutProposeNewBallot` and
	// then it will propose new `Ballot`.
	TimeoutProposeNewBallot time.Duration = time.Second * 2

	// TimeoutProposeNewBallotFull is almost same with `TimeoutProposeNewBallot`,
	// but if `Transaction`s in `TransactionPool` is over
	// `MaxTransactionsInBallot`, proposer will wait for
	// `TimeoutProposeNewBallotFull`.
	TimeoutProposeNewBallotFull time.Duration = time.Second * 1

	// TimeoutExpireRound works if running `Round` is expired for consensus.
	TimeoutExpireRound time.Duration = time.Second * 10
)
