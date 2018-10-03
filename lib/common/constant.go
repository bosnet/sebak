package common

import "time"

const (
	// BaseFee is the default transaction fee, if fee is lower than BaseFee, the
	// transaction will fail validation.
	BaseFee Amount = 10000

	// BaseReserve is minimum amount of balance for new account. By default, it
	// is `0.1` BOS.
	BaseReserve Amount = 1000000

	// GenesisBlockHeight set the block height of genesis block
	GenesisBlockHeight uint64 = 1

	// GenesisBlockConfirmedTime is the time for the confirmed time of genesis
	// block. This time is of the first commit of SEBAK.
	GenesisBlockConfirmedTime string = "2018-04-17T5:07:31.000000000Z"
)

var (
	// BallotConfirmedTimeAllowDuration is the duration time for ballot from
	// other nodes. If confirmed time of ballot has too late or ahead by
	// BallotConfirmedTimeAllowDuration, it will be considered not-wellformed.
	// For details, `Ballot.IsWellFormed()`
	BallotConfirmedTimeAllowDuration time.Duration = time.Minute * time.Duration(1)

	// MaxTransactionsInBallot limits the maximum number of `Transaction`s in
	// one proposed `Ballot`.
	MaxTransactionsInBallot int = 1000

	// MaxOperationsInTransaction limits the maximum number of `Operation`s in
	// one `Transaction`.
	MaxOperationsInTransaction int = 1000

	// DefaultInflationRatio is the default inflation ratio. If the decimal
	// points is over 17, the inflation amount will be 0, considering with
	// `MaximumBalance`.
	DefaultInflationRatio float64 = 0.00000000000000003
)
