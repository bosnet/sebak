package common

import (
	"time"

	"github.com/ulule/limiter"
)

const (
	// BaseFee is the default transaction fee, if fee is lower than BaseFee, the
	// transaction will fail validation.
	BaseFee Amount = 10000

	// BaseReserve is minimum amount of balance for new account. By default, it
	// is `0.1` BOS.
	BaseReserve Amount = 1000000

	// FrozenFee is a special transaction fee about freezing, and unfreezing.
	FrozenFee Amount = 0

	// GenesisBlockHeight set the block height of genesis block
	GenesisBlockHeight uint64 = 1

	// FirstProposedBlockHeight is used for calculating block time
	FirstProposedBlockHeight uint64 = 2

	// GenesisBlockConfirmedTime is the time for the confirmed time of genesis
	// block. This time is of the first commit of SEBAK.
	GenesisBlockConfirmedTime string = "2018-04-17T5:07:31.000000000Z"

	// InflationRatio is the inflation ratio. If the decimal points is over 17,
	// the inflation amount will be 0, considering with `MaximumBalance`. The
	// current value, `0.0000001` will increase `50BOS` in every block(current
	// genesis balance is `5000000000000000`).
	InflationRatio float64 = 0.0000001

	// BlockHeightEndOfInflation sets the block height of inflation end.
	BlockHeightEndOfInflation uint64 = 36000000

	HTTPCacheMemoryAdapterName = "mem"
	HTTPCacheRedisAdapterName  = "redis"
	HTTPCachePoolSize          = 10000

	// DefaultTxPoolLimit is the default tx pool limit.
	DefaultTxPoolLimit int = 1000000

	// DefaultOperationsInTransactionLimit is the default maximum number of
	// operations in one transaction.
	DefaultOperationsInTransactionLimit int = 1000

	// DefaultTransactionsInBallotLimit is the default maximum number of
	// transactions in one ballot.
	DefaultTransactionsInBallotLimit int = 1000

	// DefaultOperationsInBallotLimit is the default maximum number of
	// operations in one ballot. This does not count the operations of
	// `ProposerTransaction`.
	DefaultOperationsInBallotLimit int = 10000

	DefaultTimeoutINIT       = 2 * time.Second
	DefaultTimeoutSIGN       = 2 * time.Second
	DefaultTimeoutACCEPT     = 2 * time.Second
	DefaultTimeoutALLCONFIRM = 30 * time.Second
	DefaultBlockTime         = 5 * time.Second
	DefaultBlockTimeDelta    = 1 * time.Second

	// DiscoveryMessageCreatedAllowDuration limit the `DiscoveryMessage.Created`
	// is allowed or not.
	DiscoveryMessageCreatedAllowDuration time.Duration = time.Second * 10
)

var (
	// UnfreezingPeriod is the number of blocks required for unfreezing to take effect.
	// When frozen funds are unfreezed, the transaction is record in the blockchain,
	// and after `UnfreezingPeriod`, it takes effect on the account.
	// The default value, 241920, is equal to:
	// 14 (days) * 24 (hours) * 60 (minutes) * 12 (60 seconds / 5 seconds per block on average)
	UnfreezingPeriod uint64 = 241920

	// BallotConfirmedTimeAllowDuration is the duration time for ballot from
	// other nodes. If confirmed time of ballot has too late or ahead by
	// BallotConfirmedTimeAllowDuration, it will be considered not-wellformed.
	// For details, `Ballot.IsWellFormed()`
	BallotConfirmedTimeAllowDuration time.Duration = time.Minute * time.Duration(1)

	InflationRatioString string = InflationRatio2String(InflationRatio)

	// RateLimitAPI set the rate limit for API interface, the default value
	// allows 100 requests per minute.
	RateLimitAPI, _ = limiter.NewRateFromFormatted("100-M")

	// RateLimitNode set the rate limit for node interface, the default value
	// allows 100 requests per seconds.
	RateLimitNode, _ = limiter.NewRateFromFormatted("100-S")

	HTTPCacheAdapterNames = map[string]bool{
		HTTPCacheMemoryAdapterName: true,
		HTTPCacheRedisAdapterName:  true,
		"":                         true, // default value is nop cache
	}
	DefaultJSONRPCBindURL string = "http://127.0.0.1:54321/jsonrpc" // JSONRPC only can be accessed from localhost
)
