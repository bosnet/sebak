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
	TimeoutINIT   time.Duration
	TimeoutSIGN   time.Duration
	TimeoutACCEPT time.Duration
	BlockTime     time.Duration

	TxsLimit    int
	OpsLimit    int
	TxPoolLimit int

	RateLimitRuleAPI  RateLimitRule
	RateLimitRuleNode RateLimitRule
}

func NewConfig() Config {
	p := Config{}

	p.TimeoutINIT = 2 * time.Second
	p.TimeoutSIGN = 2 * time.Second
	p.TimeoutACCEPT = 2 * time.Second
	p.BlockTime = 5 * time.Second

	p.TxsLimit = 1000
	p.OpsLimit = 1000
	p.TxPoolLimit = 100000

	p.RateLimitRuleAPI = NewRateLimitRule(RateLimitAPI)
	p.RateLimitRuleNode = NewRateLimitRule(RateLimitNode)

	return p
}
