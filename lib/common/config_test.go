/*
	In this file, there are unittests for checking Config struct.
*/
package common

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

//	TestConfigSetAndGet tests setting timeout fields and checking.
func TestConfigSetAndGet(t *testing.T) {
	conf := Config{}
	conf.TimeoutINIT = 3 * time.Second
	conf.TimeoutSIGN = 1 * time.Second
	conf.TimeoutACCEPT = 1 * time.Second
	conf.TimeoutALLCONFIRM = 10 * time.Second
	conf.BlockTime = 7 * time.Second
	conf.BlockTimeDelta = 5 * time.Second

	conf.TxsLimit = 500
	conf.OpsLimit = 200

	require.Equal(t, 3*time.Second, conf.TimeoutINIT)
	require.Equal(t, 1*time.Second, conf.TimeoutSIGN)
	require.Equal(t, 1*time.Second, conf.TimeoutACCEPT)
	require.Equal(t, 10*time.Second, conf.TimeoutALLCONFIRM)
	require.Equal(t, 7*time.Second, conf.BlockTime)
	require.Equal(t, 5*time.Second, conf.BlockTimeDelta)

	require.Equal(t, 500, conf.TxsLimit)
	require.Equal(t, 200, conf.OpsLimit)
}
