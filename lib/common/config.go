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
}

func NewConfig(networkID []byte) Config {
	p := Config{}

	p.TimeoutINIT = 2 * time.Second
	p.TimeoutSIGN = 2 * time.Second
	p.TimeoutACCEPT = 2 * time.Second
	p.TimeoutALLCONFIRM = 30 * time.Second
	p.BlockTime = 5 * time.Second
	p.BlockTimeDelta = 1 * time.Second

	p.TxsLimit = DefaultTransactionsInBallotLimit
	p.OpsLimit = DefaultOperationsInTransactionLimit
	p.OpsInBallotLimit = DefaultOperationsInBallotLimit

	p.NetworkID = networkID

	p.TxPoolClientLimit = DefaultTxPoolLimit
	p.TxPoolNodeLimit = 0 // unlimited

	p.RateLimitRuleAPI = NewRateLimitRule(RateLimitAPI)
	p.RateLimitRuleNode = NewRateLimitRule(RateLimitNode)

	p.HTTPCachePoolSize = HTTPCachePoolSize
	p.JSONRPCEndpoint, _ = ParseEndpoint(DefaultJSONRPCBindURL)

	return p
}
