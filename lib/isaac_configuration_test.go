/*
	In this file, there are unittests for checking IsaacConfiguration struct.
*/
package sebak

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

//	TestConfigurationDefault tests the default timeout values.
func TestConfigurationDefault(t *testing.T) {
	n := NewIsaacConfiguration()
	require.Equal(t, n.TimeoutINIT, 2*time.Second)
	require.Equal(t, n.TimeoutSIGN, 2*time.Second)
	require.Equal(t, n.TimeoutACCEPT, 2*time.Second)
	require.Equal(t, n.TimeoutALLCONFIRM, 2*time.Second)
	require.Equal(t, uint64(1000), n.TransactionsLimit)
}

//	TestConfigurationSetAndGet tests setting timeout fields and checking.
func TestConfigurationSetAndGet(t *testing.T) {
	n := NewIsaacConfiguration()
	n.TimeoutINIT = 3 * time.Second
	n.TimeoutSIGN = 1 * time.Second
	n.TimeoutACCEPT = 1 * time.Second
	n.TimeoutALLCONFIRM = 2 * time.Second
	n.TransactionsLimit = uint64(500)

	require.Equal(t, n.TimeoutINIT, 3*time.Second)
	require.Equal(t, n.TimeoutSIGN, 1*time.Second)
	require.Equal(t, n.TimeoutACCEPT, 1*time.Second)
	require.Equal(t, n.TimeoutALLCONFIRM, 2*time.Second)
	require.Equal(t, uint64(500), n.TransactionsLimit)
}
