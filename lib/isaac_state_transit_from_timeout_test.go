// We can test that a node broadcast propose ballot or B(`EXP`) in IsaacStateManager.
// when the timeout is expired,
package sebak

import (
	"testing"
	"time"

	"boscoin.io/sebak/lib/common"

	"github.com/stretchr/testify/require"
)

// 1. All 3 Nodes.
// 2. Proposer itself.
// 3. When `IsaacStateManager` starts, the node proposes ballot to validators.
func TestStateINITProposer(t *testing.T) {
	nodeRunners := createTestNodeRunner(3)

	nr := nodeRunners[0]

	b := NewTestBroadcastor()
	nr.SetBroadcastor(b)
	nr.SetProposerCalculator(SelfProposerCalculator{})

	nr.Consensus().SetLatestConsensusedBlock(genesisBlock)

	conf := NewIsaacConfiguration()
	conf.TimeoutINIT = time.Hour
	conf.TimeoutSIGN = time.Hour
	conf.TimeoutACCEPT = time.Hour
	conf.TimeoutALLCONFIRM = time.Hour

	nr.SetConf(conf)

	nr.StartStateManager()
	time.Sleep(time.Duration(300) * time.Millisecond)

	require.Equal(t, 1, len(b.Messages))
	for _, message := range b.Messages {
		// This message must be proposed ballot
		ballot, ok := message.(Ballot)
		require.True(t, ok)
		require.Equal(t, nr.localNode.Address(), ballot.Proposer())
	}
}

// 1. All 3 Nodes.
// 2. Not proposer itself.
// 3. When `IsaacStateManager` starts, the node waits a ballot by proposer.
// 4. But TimeoutINIT is an hour, so it doesn't broadcast anything.
func TestStateINITNotProposer(t *testing.T) {
	nodeRunners := createTestNodeRunner(3)

	nr := nodeRunners[0]

	b := NewTestBroadcastor()
	nr.SetBroadcastor(b)
	nr.SetProposerCalculator(TheOtherProposerCalculator{})

	nr.Consensus().SetLatestConsensusedBlock(genesisBlock)

	conf := NewIsaacConfiguration()
	conf.TimeoutINIT = time.Hour
	conf.TimeoutSIGN = time.Hour
	conf.TimeoutACCEPT = time.Hour
	conf.TimeoutALLCONFIRM = time.Hour

	nr.SetConf(conf)

	nr.StartStateManager()
	time.Sleep(time.Duration(100) * time.Millisecond)

	require.Equal(t, 0, len(b.Messages))
}

// 1. All 3 Nodes.
// 2. Not proposer itself.
// 3. When `IsaacStateManager` starts, the node waits a ballot by proposer.
// 4. But TimeoutINIT is a millisecond.
// 5. After 200 milliseconds, the node broadcasts B(`SIGN`, `EXP`)
func TestStateINITTimeoutNotProposer(t *testing.T) {
	nodeRunners := createTestNodeRunner(3)

	nr := nodeRunners[0]

	b := NewTestBroadcastor()
	nr.SetBroadcastor(b)
	nr.SetProposerCalculator(TheOtherProposerCalculator{})
	proposer := nr.CalculateProposer(0, 0)

	require.NotEqual(t, nr.localNode.Address(), proposer)

	nr.Consensus().SetLatestConsensusedBlock(genesisBlock)

	conf := NewIsaacConfiguration()
	conf.TimeoutINIT = 1 * time.Millisecond
	conf.TimeoutSIGN = time.Hour
	conf.TimeoutACCEPT = time.Hour
	conf.TimeoutALLCONFIRM = time.Hour

	nr.SetConf(conf)

	nr.StartStateManager()
	time.Sleep(time.Duration(200) * time.Millisecond)

	require.Equal(t, 1, len(b.Messages))
	init, sign, accept := 0, 0, 0
	for _, message := range b.Messages {
		ballot, ok := message.(Ballot)
		require.True(t, ok)
		require.Equal(t, 0, len(ballot.Transactions()))
		switch ballot.State() {
		case sebakcommon.BallotStateINIT:
			init++
		case sebakcommon.BallotStateSIGN:
			sign++
			require.Equal(t, sebakcommon.VotingEXP, ballot.Vote())
		case sebakcommon.BallotStateACCEPT:
			accept++
		}
	}
	require.Equal(t, 0, init)
	require.Equal(t, 1, sign)
	require.Equal(t, 0, accept)
}

// 1. All 3 Nodes.
// 2. Proposer itself.
// 3. When `IsaacStateManager` starts, the node proposes B(`INIT`, `YES`) to validators.
// 4. Then IsaacState will be changed to `SIGN`.
// 4. But TimeoutSIGN is a millisecond.
// 5. After 200 milliseconds, the node broadcasts B(`ACCEPT`, `EXP`)
func TestStateSIGNTimeoutProposer(t *testing.T) {
	nodeRunners := createTestNodeRunner(3)

	nr := nodeRunners[0]

	b := NewTestBroadcastor()
	nr.SetBroadcastor(b)
	nr.SetProposerCalculator(SelfProposerCalculator{})
	proposer := nr.CalculateProposer(0, 0)

	require.Equal(t, nr.localNode.Address(), proposer)

	nr.Consensus().SetLatestConsensusedBlock(genesisBlock)

	conf := NewIsaacConfiguration()
	conf.TimeoutINIT = time.Hour
	conf.TimeoutSIGN = time.Millisecond
	conf.TimeoutACCEPT = time.Hour
	conf.TimeoutALLCONFIRM = time.Hour

	nr.SetConf(conf)

	nr.StartStateManager()

	time.Sleep(time.Duration(1000) * time.Millisecond)

	require.Equal(t, 2, len(b.Messages))

	init, sign, accept := 0, 0, 0
	for _, message := range b.Messages {
		ballot, ok := message.(Ballot)
		require.True(t, ok)
		require.Equal(t, 0, len(ballot.Transactions()))
		switch ballot.State() {
		case sebakcommon.BallotStateINIT:
			init++
			require.Equal(t, sebakcommon.VotingYES, ballot.Vote())
		case sebakcommon.BallotStateSIGN:
			sign++
			require.Equal(t, sebakcommon.VotingEXP, ballot.Vote())
		case sebakcommon.BallotStateACCEPT:
			accept++
			require.Equal(t, sebakcommon.VotingEXP, ballot.Vote())
		}
	}
	require.Equal(t, 1, init)
	require.Equal(t, 0, sign)
	require.Equal(t, 1, accept)
}

// 1. All 3 Nodes.
// 2. Not proposer itself.
// 3. When `IsaacStateManager` starts, the node waits a ballot by proposer.
// 4. TimeoutINIT is a millisecond.
// 5. After milliseconds, the node broadcasts B(`SIGN`, `EXP`).
// 6. IsaacState is changed to `SIGN`.
// 7. TimeoutSIGN is a millisecond.
// 8. After milliseconds, the node broadcasts B(`ACCEPT`, `EXP`).
func TestStateSIGNTimeoutNotProposer(t *testing.T) {
	nodeRunners := createTestNodeRunner(3)

	nr := nodeRunners[0]

	b := NewTestBroadcastor()
	nr.SetBroadcastor(b)
	nr.SetProposerCalculator(TheOtherProposerCalculator{})
	proposer := nr.CalculateProposer(0, 0)

	require.NotEqual(t, nr.localNode.Address(), proposer)

	nr.Consensus().SetLatestConsensusedBlock(genesisBlock)

	conf := NewIsaacConfiguration()
	conf.TimeoutINIT = time.Millisecond
	conf.TimeoutSIGN = time.Millisecond
	conf.TimeoutACCEPT = time.Hour
	conf.TimeoutALLCONFIRM = time.Hour

	nr.SetConf(conf)

	nr.StartStateManager()
	time.Sleep(time.Duration(300) * time.Millisecond)

	require.Equal(t, 2, len(b.Messages))
	init, sign, accept := 0, 0, 0
	for _, message := range b.Messages {
		ballot, ok := message.(Ballot)
		require.True(t, ok)
		require.Equal(t, 0, len(ballot.Transactions()))
		switch ballot.State() {
		case sebakcommon.BallotStateINIT:
			init++
			require.Equal(t, sebakcommon.VotingYES, ballot.Vote())
		case sebakcommon.BallotStateSIGN:
			sign++
			require.Equal(t, sebakcommon.VotingEXP, ballot.Vote())
		case sebakcommon.BallotStateACCEPT:
			accept++
			require.Equal(t, sebakcommon.VotingEXP, ballot.Vote())
		}
	}
	require.Equal(t, 0, init)
	require.Equal(t, 1, sign)
	require.Equal(t, 1, accept)
}

// 1. All 3 Nodes.
// 2. Proposer itself at round 0.
// 3. When `IsaacStateManager` starts, the node proposes a ballot.
// 4. IsaacState is changed to `SIGN`.
// 5. TimeoutSIGN is a millisecond.
// 6. After milliseconds, the node broadcasts B(`ACCEPT`, `EXP`).
func TestStateACCEPTTimeoutProposerThenNotProposer(t *testing.T) {
	nodeRunners := createTestNodeRunner(3)

	nr := nodeRunners[0]

	b := NewTestBroadcastor()
	nr.SetBroadcastor(b)
	nr.SetProposerCalculator(&SelfProposerThenNotProposer{})

	proposer := nr.CalculateProposer(0, 0)
	require.Equal(t, nr.localNode.Address(), proposer)

	proposer = nr.CalculateProposer(0, 1)
	require.NotEqual(t, nr.localNode.Address(), proposer)

	nr.Consensus().SetLatestConsensusedBlock(genesisBlock)

	conf := NewIsaacConfiguration()
	conf.TimeoutINIT = time.Hour
	conf.TimeoutSIGN = time.Millisecond
	conf.TimeoutACCEPT = time.Millisecond
	conf.TimeoutALLCONFIRM = time.Hour

	nr.SetConf(conf)

	nr.StartStateManager()
	time.Sleep(time.Duration(300) * time.Millisecond)

	require.Equal(t, 2, len(b.Messages))
	init, sign, accept := 0, 0, 0
	for _, message := range b.Messages {
		ballot, ok := message.(Ballot)
		require.True(t, ok)
		require.Equal(t, 0, len(ballot.Transactions()))
		switch ballot.State() {
		case sebakcommon.BallotStateINIT:
			init++
			require.Equal(t, ballot.Vote(), sebakcommon.VotingYES)
		case sebakcommon.BallotStateSIGN:
			sign++
		case sebakcommon.BallotStateACCEPT:
			accept++
			require.Equal(t, ballot.Vote(), sebakcommon.VotingEXP)
		}
	}

	require.Equal(t, 1, init)
	require.Equal(t, 0, sign)
	require.Equal(t, 1, accept)
}
