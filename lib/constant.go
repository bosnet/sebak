package sebak

import (
	"time"

	"boscoin.io/sebak/lib/common"
)

const (
	// BaseFee is the default transaction fee, if fee is lower than BaseFee, the
	// transaction will fail validation.
	BaseFee sebakcommon.Amount = 10000
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

	// MaxTransactionsInBallot limits the maximum number of `Transaction`s in
	// one proposed `Ballot`.
	MaxTransactionsInBallot int = 1000

	// MaxOperationsInTransaction limits the maximum number of `Operation`s in
	// one `Transaction`.
	MaxOperationsInTransaction int = 200
)
