package runner

import (
	"testing"
	"time"

	"boscoin.io/sebak/lib/consensus"
	"github.com/stretchr/testify/require"
)

func TestConnectionManagerBroadcaster(t *testing.T) {
	conf := consensus.NewISAACConfiguration()
	conf.TimeoutALLCONFIRM = 1 * time.Millisecond

	nodeRunners := createTestNodeRunner(3, conf)

	nr := nodeRunners[0]

	recv := make(chan struct{})

	b := NewTestBroadcaster(recv)
	nr.Consensus().SetBroadcaster(b)
	nr.Consensus().ConnectionManager().SetProposerCalculator(SelfProposerCalculator{
		nodeRunner: nr,
	})

	nr.Consensus().SetLatestConsensusedBlock(genesisBlock)

	nr.StartStateManager()
	defer nr.StopStateManager()

	<-recv
	require.Equal(t, 1, len(b.Messages()))
}
