package runner

import (
	"testing"

	"github.com/stretchr/testify/require"

	"boscoin.io/sebak/lib/ballot"
	"boscoin.io/sebak/lib/consensus"
)

// We can test the transition of ballotState in SelfMode.
func TestSelfMode(t *testing.T) {
	conf := consensus.NewISAACConfiguration()
	conf.BlockTime = 0

	nr, _ := createSelfModeNodeRunner(conf, nil)

	isaac := nr.Consensus()
	isaac.SetLatestConsensusedBlock(genesisBlock)

	recv := make(chan struct{})
	nr.isaacStateManager.SetTransitSignal(func() {
		recv <- struct{}{}
	})

	nr.StartStateManager()
	defer nr.StopStateManager()

	<-recv

	require.Equal(t, ballot.StateALLCONFIRM, nr.isaacStateManager.State().BallotState)
	require.Equal(t, uint64(2), isaac.LatestConfirmedBlock().Height)

	<-recv

	require.Equal(t, ballot.StateALLCONFIRM, nr.isaacStateManager.State().BallotState)
	require.Equal(t, uint64(3), isaac.LatestConfirmedBlock().Height)

	<-recv

	require.Equal(t, ballot.StateALLCONFIRM, nr.isaacStateManager.State().BallotState)
	require.Equal(t, uint64(4), isaac.LatestConfirmedBlock().Height)
}
