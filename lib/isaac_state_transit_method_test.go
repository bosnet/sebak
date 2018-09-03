// We can check that the `TransitIsaacState()` call transitions the state.
package sebak

import (
	"testing"
	"time"

	"boscoin.io/sebak/lib/common"

	"github.com/stretchr/testify/require"
)

// 1. All 3 Nodes.
// 2. Not proposer itself.
// 3. When `IsaacStateManager` starts, the node waits for a proposed ballot.
// 4. TransitIsaacState(SIGN) method is called.
// 5. IsaacState is changed to `SIGN`.
// 6. TimeoutSIGN is a millisecond.
// 7. After milliseconds, the node broadcasts B(`ACCEPT`, `EXP`).
func TestStateTransitFromTimeoutInitToAccept(t *testing.T) {
	nodeRunners := createTestNodeRunner(3)
	nr := nodeRunners[0]

	b := NewTestBroadcastor()
	nr.SetBroadcastor(b)
	nr.SetProposerCalculator(TheOtherProposerCalculator{})

	nr.Consensus().SetLatestConsensusedBlock(genesisBlock)

	conf := NewIsaacConfiguration()
	conf.TimeoutINIT = time.Hour
	conf.TimeoutSIGN = time.Millisecond
	conf.TimeoutACCEPT = time.Millisecond
	conf.TimeoutALLCONFIRM = time.Millisecond

	nr.SetConf(conf)

	nr.StartStateManager()
	time.Sleep(time.Duration(100) * time.Millisecond)
	require.Equal(t, 0, len(b.Messages))
	for _, message := range b.Messages {
		ballot, ok := message.(Ballot)
		require.True(t, ok)
		require.Equal(t, nr.localNode.Address(), ballot.Proposer())
		require.Equal(t, sebakcommon.BallotStateINIT, ballot.State())
		require.Equal(t, sebakcommon.VotingYES, ballot.Vote())
	}

	nr.TransitIsaacState(nr.nodeRunnerStateManager.State().round, sebakcommon.BallotStateSIGN)
	time.Sleep(time.Duration(100) * time.Millisecond)
	require.Equal(t, 1, len(b.Messages))
	for _, message := range b.Messages {
		ballot, ok := message.(Ballot)
		require.True(t, ok)
		require.Equal(t, nr.localNode.Address(), ballot.Proposer())
		require.Equal(t, sebakcommon.BallotStateACCEPT, ballot.State())
		require.Equal(t, sebakcommon.VotingEXP, ballot.Vote())
	}
}

// 1. All 3 Nodes.
// 1. Proposer itself.
// 1. When `IsaacStateManager` starts, the node proposes a ballot.
// 1. IsaacState is changed to `SIGN`.
// 1. TransitIsaacState(ACCEPT) method is called.
// 1. IsaacState is changed to `ACCEPT`.
// 1. TimeoutACCEPT is a millisecond.
// 1. After milliseconds, IsaacState is back to `INIT`
func TestStateTransitFromTimeoutSignToAccept(t *testing.T) {
	nodeRunners := createTestNodeRunner(3)
	nr := nodeRunners[0]

	b := NewTestBroadcastor()
	nr.SetBroadcastor(b)
	nr.SetProposerCalculator(SelfProposerCalculator{})

	nr.Consensus().SetLatestConsensusedBlock(genesisBlock)

	conf := NewIsaacConfiguration()
	conf.TimeoutINIT = time.Hour
	conf.TimeoutSIGN = time.Hour
	conf.TimeoutACCEPT = time.Millisecond
	conf.TimeoutALLCONFIRM = time.Millisecond

	nr.SetConf(conf)

	nr.StartStateManager()
	time.Sleep(time.Duration(200) * time.Millisecond)

	require.Equal(t, sebakcommon.BallotStateSIGN, nr.nodeRunnerStateManager.State().ballotState)
	require.Equal(t, 1, len(b.Messages))
	for _, message := range b.Messages {
		ballot, ok := message.(Ballot)
		require.True(t, ok)
		require.Equal(t, nr.localNode.Address(), ballot.Proposer())
		require.Equal(t, sebakcommon.BallotStateINIT, ballot.State())
		require.Equal(t, sebakcommon.VotingYES, ballot.Vote())
	}

	nr.TransitIsaacState(nr.nodeRunnerStateManager.State().round, sebakcommon.BallotStateACCEPT)
	time.Sleep(time.Duration(200) * time.Millisecond)
	require.Equal(t, 2, len(b.Messages))
	for _, message := range b.Messages {
		ballot, ok := message.(Ballot)
		require.True(t, ok)
		require.Equal(t, nr.localNode.Address(), ballot.Proposer())
		require.Equal(t, sebakcommon.BallotStateINIT, ballot.State())
		require.Equal(t, sebakcommon.VotingYES, ballot.Vote())
	}
}
