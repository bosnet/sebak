package sebak

import (
	"time"

	"boscoin.io/sebak/lib/common"
)

const (
	// BaseFee is the default transaction fee, if fee is lower than BaseFee, the
	// transaction will fail validation.
	BaseFee sebakcommon.Amount = 10000

	// BallotConfirmedTimeAllowDuration is the duration time for ballot from
	// other nodes. If confirmed time of ballot has too late or ahead by
	// BallotConfirmedTimeAllowDuration, it will be considered not-wellformed.
	// For details, `Ballot.IsWellFormed()`
	BallotConfirmedTimeAllowDuration time.Duration = time.Minute * time.Duration(1)
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
)
