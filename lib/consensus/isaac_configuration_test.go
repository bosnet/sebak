/*
	In this file, there are unittests for checking ISAACConfiguration struct.
*/
package consensus

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

//	TestConfigurationDefault tests the default timeout values.
func TestConfigurationDefault(t *testing.T) {
	n := NewISAACConfiguration()
	require.Equal(t, n.TimeoutINIT, 2*time.Second)
	require.Equal(t, n.TimeoutSIGN, 2*time.Second)
	require.Equal(t, n.TimeoutACCEPT, 2*time.Second)
	require.Equal(t, n.BlockTime, 5*time.Second)
	require.Equal(t, uint64(1000), n.TransactionsLimit)
}

//	TestConfigurationSetAndGet tests setting timeout fields and checking.
func TestConfigurationSetAndGet(t *testing.T) {
	n := NewISAACConfiguration()
	n.TimeoutINIT = 3 * time.Second
	n.TimeoutSIGN = 1 * time.Second
	n.TimeoutACCEPT = 1 * time.Second
	n.BlockTime = 7 * time.Second
	n.TransactionsLimit = uint64(500)

	require.Equal(t, n.TimeoutINIT, 3*time.Second)
	require.Equal(t, n.TimeoutSIGN, 1*time.Second)
	require.Equal(t, n.TimeoutACCEPT, 1*time.Second)
	require.Equal(t, n.BlockTime, 7*time.Second)
	require.Equal(t, uint64(500), n.TransactionsLimit)
}
