// Provide test utilities for the common package
package common

import (
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/rlp"
	"github.com/stretchr/testify/require"
)

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
	p.InitialBalance = MaximumBalance

	p.TxPoolClientLimit = DefaultTxPoolLimit
	p.TxPoolNodeLimit = 0 // unlimited

	p.RateLimitRuleAPI = NewRateLimitRule(RateLimitAPI)
	p.RateLimitRuleNode = NewRateLimitRule(RateLimitNode)

	p.HTTPCachePoolSize = HTTPCachePoolSize

	return p
}

// Test that a record that gets serialized to RLP deserialize to the same data
//
// Params:
//   t = The testing object
//   record = A pointer to the record to serialize
func CheckRoundTripRLP(t *testing.T, record interface{}) {
	binary, err := rlp.EncodeToBytes(record)
	require.NoError(t, err)

	result := reflect.New(reflect.TypeOf(record))
	err = rlp.DecodeBytes(binary, result.Interface())
	require.NoError(t, err)

	require.Equal(t, record, result.Elem().Interface())
	require.Equal(t, MustMakeObjectHash(record), MustMakeObjectHash(result.Elem().Interface()))
}

// Utility to get a new `common.Endpoint` from a string
func MustParseEndpoint(endpoint string) *Endpoint {
	if ret, err := ParseEndpoint(endpoint); err != nil {
		panic(err)
	} else {
		return ret
	}
}
