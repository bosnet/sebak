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

	// GenesisBlockHeight set the block height of genesis block
	GenesisBlockHeight uint64 = 1

	// FirstConsensusBlockHeight is used for calculating block time
	FirstConsensusBlockHeight uint64 = 2

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
	RateLimitAPI limiter.Rate = limiter.Rate{
		Period: 1 * time.Minute,
		Limit:  100,
	}

	// RateLimitNode set the rate limit for node interface, the default value
	// allows 100 requests per seconds.
	RateLimitNode limiter.Rate = limiter.Rate{
		Period: 1 * time.Second,
		Limit:  100,
	}

	HTTPCacheAdapterNames = map[string]bool{
		HTTPCacheMemoryAdapterName: true,
		HTTPCacheRedisAdapterName:  true,
		"":                         true, // default value is nop cache
	}
)
