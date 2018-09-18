package runner

import (
	"testing"

	"boscoin.io/sebak/lib/consensus"
	"github.com/stretchr/testify/require"
)

func TestConnectionManagerBroadcaster(t *testing.T) {
	conf := consensus.NewISAACConfiguration()

	recv := make(chan struct{})
	nr, _, cm := createNodeRunnerForTesting(3, conf, recv)

	nr.Consensus().SetLatestConsensusedBlock(genesisBlock)

	nr.StartStateManager()
	defer nr.StopStateManager()

	<-recv
	require.Equal(t, 1, len(cm.Messages()))
}
