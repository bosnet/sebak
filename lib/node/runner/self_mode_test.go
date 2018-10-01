package runner

import (
	"testing"

	"github.com/stretchr/testify/require"

	"boscoin.io/sebak/lib/ballot"
)

// We can test the transition of ballotState in SelfMode.
func TestSelfMode(t *testing.T) {
	nodeRunners, _ := createTestNodeRunnersHTTP2NetworkWithReady(1)

	nr := nodeRunners[0]

	nr.isaacStateManager.Conf.BlockTime = 0
	validators := nr.ConnectionManager().AllValidators()
	require.Equal(t, 1, len(validators))
	require.Equal(t, nr.localNode.Address(), validators[0])

	isaac := nr.Consensus()
	// isaac.SetLatestConsensusedBlock(genesisBlock)

	recv := make(chan struct{})
	nr.isaacStateManager.SetTransitSignal(func() {
		recv <- struct{}{}
	})

	<-recv
	require.Equal(t, ballot.StateINIT, nr.isaacStateManager.State().BallotState)

	<-recv
	require.Equal(t, ballot.StateSIGN, nr.isaacStateManager.State().BallotState)

	<-recv
	require.Equal(t, ballot.StateACCEPT, nr.isaacStateManager.State().BallotState)

	<-recv
	require.Equal(t, ballot.StateALLCONFIRM, nr.isaacStateManager.State().BallotState)
	require.Equal(t, uint64(2), isaac.LatestConfirmedBlock().Height)

	<-recv
	require.Equal(t, ballot.StateINIT, nr.isaacStateManager.State().BallotState)

	<-recv
	require.Equal(t, ballot.StateSIGN, nr.isaacStateManager.State().BallotState)

	<-recv
	require.Equal(t, ballot.StateACCEPT, nr.isaacStateManager.State().BallotState)

	<-recv
	require.Equal(t, ballot.StateALLCONFIRM, nr.isaacStateManager.State().BallotState)
	require.Equal(t, uint64(3), isaac.LatestConfirmedBlock().Height)
}
