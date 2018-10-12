package runner

import (
	"testing"
	"time"

	"boscoin.io/sebak/lib/block"
	"github.com/stretchr/testify/require"
)

func TestBlockTime(t *testing.T) {
	nodeRunners, _ := createTestNodeRunnersHTTP2NetworkWithReady(3)
	defer func() {
		for _, nr := range nodeRunners {
			nr.Stop()
		}
	}()

	nr := nodeRunners[0]

	sec := 60 * time.Second

	time.Sleep(sec)

	latestBlock := nr.Consensus().LatestBlock()
	latestHeight := latestBlock.Height
	for i := 0; i < int(latestHeight); i++ {
		b, err := block.GetBlockByHeight(nr.Storage(), uint64(i+1))
		require.Nil(t, err)
		t.Log(b.Header.Timestamp.String())
	}

	genesis, err := block.GetBlockByHeight(nr.Storage(), uint64(1))
	require.Nil(t, err)
	averageBlockTime := latestBlock.Header.Timestamp.Sub(genesis.Header.Timestamp) / time.Duration(latestHeight-1)

	t.Log("averageBlockTime", averageBlockTime)
	require.True(t, averageBlockTime >= 4500*time.Millisecond)
	require.True(t, averageBlockTime <= 5500*time.Millisecond)
}
