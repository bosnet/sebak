package sebak

import (
	"testing"
	"time"

	"boscoin.io/sebak/lib/common"

	"github.com/stretchr/testify/require"
)

func TestStateINITProposer(t *testing.T) {
	nodeRunners := createTestNodeRunner(3)

	nr := nodeRunners[0]

	b := NewTestBroadcastor()
	nr.SetBroadcastor(b)
	nr.SetProposerCalculator(SelfProposerCalculator{})

	nr.Consensus().SetLatestConsensusedBlock(genesisBlock)

	conf := NewNodeRunnerConfiguration()
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

func TestStateINITNotProposer(t *testing.T) {
	nodeRunners := createTestNodeRunner(3)

	nr := nodeRunners[0]

	b := NewTestBroadcastor()
	nr.SetBroadcastor(b)
	nr.SetProposerCalculator(TheOtherProposerCalculator{})

	nr.Consensus().SetLatestConsensusedBlock(genesisBlock)

	conf := NewNodeRunnerConfiguration()
	conf.TimeoutINIT = time.Hour
	conf.TimeoutSIGN = time.Hour
	conf.TimeoutACCEPT = time.Hour
	conf.TimeoutALLCONFIRM = time.Hour

	nr.SetConf(conf)

	nr.StartStateManager()
	time.Sleep(time.Duration(100) * time.Millisecond)

	require.Equal(t, 0, len(b.Messages))
}

func TestStateINITTimeoutNotProposer(t *testing.T) {
	nodeRunners := createTestNodeRunner(3)

	nr := nodeRunners[0]

	b := NewTestBroadcastor()
	nr.SetBroadcastor(b)
	nr.SetProposerCalculator(TheOtherProposerCalculator{})
	proposer := nr.CalculateProposer(0, 0)

	require.NotEqual(t, nr.localNode.Address(), proposer)

	nr.Consensus().SetLatestConsensusedBlock(genesisBlock)

	conf := NewNodeRunnerConfiguration()
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

func TestStateSIGNTimeoutProposer(t *testing.T) {
	nodeRunners := createTestNodeRunner(3)

	nr := nodeRunners[0]

	b := NewTestBroadcastor()
	nr.SetBroadcastor(b)
	nr.SetProposerCalculator(SelfProposerCalculator{})
	proposer := nr.CalculateProposer(0, 0)

	require.Equal(t, nr.localNode.Address(), proposer)

	nr.Consensus().SetLatestConsensusedBlock(genesisBlock)

	conf := NewNodeRunnerConfiguration()
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

func TestStateSIGNTimeoutNotProposer(t *testing.T) {
	nodeRunners := createTestNodeRunner(3)

	nr := nodeRunners[0]

	b := NewTestBroadcastor()
	nr.SetBroadcastor(b)
	nr.SetProposerCalculator(TheOtherProposerCalculator{})
	proposer := nr.CalculateProposer(0, 0)

	require.NotEqual(t, nr.localNode.Address(), proposer)

	nr.Consensus().SetLatestConsensusedBlock(genesisBlock)

	conf := NewNodeRunnerConfiguration()
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

	conf := NewNodeRunnerConfiguration()
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

func TestStateTransitFromTimeoutInitToAccept(t *testing.T) {
	nodeRunners := createTestNodeRunner(3)
	nr := nodeRunners[0]

	b := NewTestBroadcastor()
	nr.SetBroadcastor(b)
	nr.SetProposerCalculator(TheOtherProposerCalculator{})

	nr.Consensus().SetLatestConsensusedBlock(genesisBlock)

	conf := NewNodeRunnerConfiguration()
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

	nr.TransitNodeRunnerState(nr.nodeRunnerStateManager.State().round, sebakcommon.BallotStateSIGN)
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

func TestStateTransitFromTimeoutSignToAccept(t *testing.T) {
	nodeRunners := createTestNodeRunner(3)
	nr := nodeRunners[0]

	b := NewTestBroadcastor()
	nr.SetBroadcastor(b)
	nr.SetProposerCalculator(SelfProposerCalculator{})

	nr.Consensus().SetLatestConsensusedBlock(genesisBlock)

	conf := NewNodeRunnerConfiguration()
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

	nr.TransitNodeRunnerState(nr.nodeRunnerStateManager.State().round, sebakcommon.BallotStateACCEPT)
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
