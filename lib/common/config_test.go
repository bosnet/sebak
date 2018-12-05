/*
	In this file, there are unittests for checking Config struct.
*/
package common

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

//	TestConfigDefault tests the default timeout values.
func TestConfigDefault(t *testing.T) {
	n := NewTestConfig()
	require.Equal(t, DefaultTimeoutINIT, n.TimeoutINIT)
	require.Equal(t, DefaultTimeoutSIGN, n.TimeoutSIGN)
	require.Equal(t, DefaultTimeoutACCEPT, n.TimeoutACCEPT)
	require.Equal(t, DefaultTimeoutALLCONFIRM, n.TimeoutALLCONFIRM)
	require.Equal(t, DefaultBlockTime, n.BlockTime)
	require.Equal(t, DefaultBlockTimeDelta, n.BlockTimeDelta)

	require.Equal(t, DefaultTransactionsInBallotLimit, n.TxsLimit)
	require.Equal(t, DefaultOperationsInTransactionLimit, n.OpsLimit)
}

//	TestConfigSetAndGet tests setting timeout fields and checking.
func TestConfigSetAndGet(t *testing.T) {
	n := NewTestConfig()
	n.TimeoutINIT = 3 * time.Second
	n.TimeoutSIGN = 1 * time.Second
	n.TimeoutACCEPT = 1 * time.Second
	n.TimeoutALLCONFIRM = 10 * time.Second
	n.BlockTime = 7 * time.Second
	n.BlockTimeDelta = 5 * time.Second

	n.TxsLimit = 500
	n.OpsLimit = 200

	require.Equal(t, 3*time.Second, n.TimeoutINIT)
	require.Equal(t, 1*time.Second, n.TimeoutSIGN)
	require.Equal(t, 1*time.Second, n.TimeoutACCEPT)
	require.Equal(t, 10*time.Second, n.TimeoutALLCONFIRM)
	require.Equal(t, 7*time.Second, n.BlockTime)
	require.Equal(t, 5*time.Second, n.BlockTimeDelta)

	require.Equal(t, 500, n.TxsLimit)
	require.Equal(t, 200, n.OpsLimit)
}
