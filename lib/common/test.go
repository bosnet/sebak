// Provide test utilities for the common package
package common

// Initialize a new config object for unittests
func NewTestConfig() Config {
	p := Config{}

	p.TimeoutINIT = DefaultTimeoutINIT
	p.TimeoutSIGN = DefaultTimeoutSIGN
	p.TimeoutACCEPT = DefaultTimeoutACCEPT
	p.TimeoutALLCONFIRM = DefaultTimeoutALLCONFIRM
	p.BlockTime = 0
	p.BlockTimeDelta = DefaultBlockTimeDelta

	p.TxsLimit = DefaultTransactionsInBallotLimit
	p.OpsLimit = DefaultOperationsInTransactionLimit
	p.OpsInBallotLimit = DefaultOperationsInBallotLimit

	p.NetworkID = []byte("sebak-unittest")

	p.TxPoolClientLimit = DefaultTxPoolLimit
	p.TxPoolNodeLimit = 0 // unlimited

	p.RateLimitRuleAPI = NewRateLimitRule(RateLimitAPI)
	p.RateLimitRuleNode = NewRateLimitRule(RateLimitNode)

	p.HTTPCachePoolSize = HTTPCachePoolSize

	return p
}
