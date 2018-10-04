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
	n := NewConfig()
	require.Equal(t, 2*time.Second, n.TimeoutINIT)
	require.Equal(t, 2*time.Second, n.TimeoutSIGN)
	require.Equal(t, 2*time.Second, n.TimeoutACCEPT)
	require.Equal(t, 5*time.Second, n.BlockTime)

	require.Equal(t, 1000, n.TxsLimit)
	require.Equal(t, 1000, n.OpsLimit)
}

//	TestConfigSetAndGet tests setting timeout fields and checking.
func TestConfigSetAndGet(t *testing.T) {
	n := NewConfig()
	n.TimeoutINIT = 3 * time.Second
	n.TimeoutSIGN = 1 * time.Second
	n.TimeoutACCEPT = 1 * time.Second
	n.BlockTime = 7 * time.Second

	n.TxsLimit = 500
	n.OpsLimit = 200

	require.Equal(t, 3*time.Second, n.TimeoutINIT)
	require.Equal(t, 1*time.Second, n.TimeoutSIGN)
	require.Equal(t, 1*time.Second, n.TimeoutACCEPT)
	require.Equal(t, 7*time.Second, n.BlockTime)

	require.Equal(t, 500, n.TxsLimit)
	require.Equal(t, 200, n.OpsLimit)
}
