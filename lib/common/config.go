package common

import (
	"time"
)

//
// Config has timeout features and transaction limit.
// The Config is included in ISAACStateManager and
// these timeout features are used in ISAAC consensus.
//
type Config struct {
	TimeoutINIT       time.Duration
	TimeoutSIGN       time.Duration
	TimeoutACCEPT     time.Duration
	TimeoutALLCONFIRM time.Duration
	BlockTime         time.Duration
	BlockTimeDelta    time.Duration

	TxsLimit          int
	OpsLimit          int
	OpsInBallotLimit  int
	TxPoolClientLimit int
	TxPoolNodeLimit   int

	NetworkID []byte

	// Those fields are not consensus-related
	RateLimitRuleAPI  RateLimitRule
	RateLimitRuleNode RateLimitRule

	HTTPCacheAdapter    string
	HTTPCachePoolSize   int
	HTTPCacheRedisAddrs map[string]string

	CongressAccountAddress string
	CommonAccountAddress   string

	JSONRPCEndpoint *Endpoint

	WatcherMode bool
}

func NewConfig(networkID []byte) Config {
	p := Config{}

	p.TimeoutINIT = DefaultTimeoutINIT
	p.TimeoutSIGN = DefaultTimeoutSIGN
	p.TimeoutACCEPT = DefaultTimeoutACCEPT
	p.TimeoutALLCONFIRM = DefaultTimeoutALLCONFIRM
	p.BlockTime = DefaultBlockTime
	p.BlockTimeDelta = DefaultBlockTimeDelta

	p.TxsLimit = DefaultTransactionsInBallotLimit
	p.OpsLimit = DefaultOperationsInTransactionLimit
	p.OpsInBallotLimit = DefaultOperationsInBallotLimit

	p.NetworkID = networkID

	p.TxPoolClientLimit = DefaultTxPoolLimit
	p.TxPoolNodeLimit = 0 // unlimited

	p.RateLimitRuleAPI = NewRateLimitRule(RateLimitAPI)
	p.RateLimitRuleNode = NewRateLimitRule(RateLimitNode)

	p.HTTPCachePoolSize = HTTPCachePoolSize

	return p
}
