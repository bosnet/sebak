// We can test that a node broadcast propose ballot or B(`EXP`) in ISAACStateManager.
package runner

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"boscoin.io/sebak/lib/ballot"
	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/consensus"
)

// 1. All 3 Nodes.
// 2. Proposer itself.
// 3. When `ISAACStateManager` starts, the node proposes ballot to validators.
func TestStateINITProposer(t *testing.T) {
	conf := consensus.NewISAACConfiguration()
	conf.TimeoutINIT = time.Hour
	conf.TimeoutSIGN = time.Hour
	conf.TimeoutACCEPT = time.Hour
	conf.TimeoutALLCONFIRM = time.Hour

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
	for _, message := range b.Messages() {
		// This message must be proposed ballot
		b, ok := message.(block.Ballot)
		require.True(t, ok)
		require.Equal(t, nr.localNode.Address(), b.Proposer())
	}
}

// 1. All 3 Nodes.
// 2. Not proposer itself.
// 3. When `ISAACStateManager` starts, the node waits a ballot by proposer.
// 4. But TimeoutINIT is an hour, so it doesn't broadcast anything.
func TestStateINITNotProposer(t *testing.T) {
	conf := consensus.NewISAACConfiguration()
	conf.TimeoutINIT = time.Hour
	conf.TimeoutSIGN = time.Hour
	conf.TimeoutACCEPT = time.Hour
	conf.TimeoutALLCONFIRM = time.Hour

	nodeRunners := createTestNodeRunner(3, conf)

	nr := nodeRunners[0]

	b := NewTestBroadcaster(nil)
	nr.Consensus().SetBroadcaster(b)
	nr.Consensus().ConnectionManager().SetProposerCalculator(TheOtherProposerCalculator{
		nodeRunner: nr,
	})

	nr.Consensus().SetLatestConsensusedBlock(genesisBlock)

	nr.StartStateManager()
	defer nr.StopStateManager()
	time.Sleep(1 * time.Second)

	require.Equal(t, 0, len(b.Messages()))
}

// 1. All 3 Nodes.
// 2. Not proposer itself.
// 3. When `ISAACStateManager` starts, the node waits a ballot by proposer.
// 4. But TimeoutINIT is a millisecond.
// 5. After timeout, the node broadcasts B(`SIGN`, `EXP`)
func TestStateINITTimeoutNotProposer(t *testing.T) {
	conf := consensus.NewISAACConfiguration()
	conf.TimeoutINIT = 200 * time.Millisecond
	conf.TimeoutSIGN = time.Hour
	conf.TimeoutACCEPT = time.Hour
	conf.TimeoutALLCONFIRM = time.Hour

	nodeRunners := createTestNodeRunner(3, conf)

	nr := nodeRunners[0]

	recv := make(chan struct{})
	b := NewTestBroadcaster(recv)
	nr.Consensus().SetBroadcaster(b)
	nr.Consensus().ConnectionManager().SetProposerCalculator(TheOtherProposerCalculator{
		nodeRunner: nr,
	})
	proposer := nr.Consensus().ConnectionManager().CalculateProposer(0, 0)

	require.NotEqual(t, nr.localNode.Address(), proposer)

	nr.Consensus().SetLatestConsensusedBlock(genesisBlock)

	nr.StartStateManager()
	defer nr.StopStateManager()
	require.Equal(t, ballot.StateINIT, nr.isaacStateManager.State().BallotState)

	<-recv
	require.Equal(t, ballot.StateSIGN, nr.isaacStateManager.State().BallotState)

	require.Equal(t, 1, len(b.Messages()))
	init, sign, accept := 0, 0, 0
	for _, message := range b.Messages() {
		b, ok := message.(block.Ballot)
		require.True(t, ok)
		require.Equal(t, 0, len(b.Transactions()))
		switch b.State() {
		case ballot.StateINIT:
			init++
		case ballot.StateSIGN:
			sign++
			require.Equal(t, ballot.VotingEXP, b.Vote())
		case ballot.StateACCEPT:
			accept++
		}
	}
	require.Equal(t, 0, init)
	require.Equal(t, 1, sign)
	require.Equal(t, 0, accept)
}

// 1. All 3 Nodes.
// 2. Proposer itself.
// 3. When `ISAACStateManager` starts, the node proposes B(`INIT`, `YES`) to validators.
// 4. Then ISAACState will be changed to `SIGN`.
// 5. But TimeoutSIGN is a millisecond.
// 6. After timeout, the node broadcasts B(`ACCEPT`, `EXP`)
func TestStateSIGNTimeoutProposer(t *testing.T) {
	conf := consensus.NewISAACConfiguration()
	conf.TimeoutINIT = time.Hour
	conf.TimeoutSIGN = 200 * time.Millisecond
	conf.TimeoutACCEPT = time.Hour
	conf.TimeoutALLCONFIRM = time.Hour

	nodeRunners := createTestNodeRunner(3, conf)

	nr := nodeRunners[0]

	recv := make(chan struct{})
	b := NewTestBroadcaster(recv)
	nr.Consensus().SetBroadcaster(b)
	nr.Consensus().ConnectionManager().SetProposerCalculator(SelfProposerCalculator{
		nodeRunner: nr,
	})
	proposer := nr.Consensus().ConnectionManager().CalculateProposer(0, 0)

	require.Equal(t, nr.localNode.Address(), proposer)

	nr.Consensus().SetLatestConsensusedBlock(genesisBlock)

	nr.StartStateManager()
	defer nr.StopStateManager()

	require.Equal(t, ballot.StateINIT, nr.isaacStateManager.State().BallotState)

	<-recv
	require.Equal(t, 1, len(b.Messages()))

	<-recv
	require.Equal(t, ballot.StateACCEPT, nr.isaacStateManager.State().BallotState)
	require.Equal(t, 2, len(b.Messages()))

	init, sign, accept := 0, 0, 0
	for _, message := range b.Messages() {
		b, ok := message.(block.Ballot)
		require.True(t, ok)
		require.Equal(t, 0, len(b.Transactions()))
		switch b.State() {
		case ballot.StateINIT:
			init++
			require.Equal(t, ballot.VotingYES, b.Vote())
		case ballot.StateSIGN:
			sign++
			require.Equal(t, ballot.VotingEXP, b.Vote())
		case ballot.StateACCEPT:
			accept++
			require.Equal(t, ballot.VotingEXP, b.Vote())
		}
	}
	require.Equal(t, 1, init)
	require.Equal(t, 0, sign)
	require.Equal(t, 1, accept)
}

// 1. All 3 Nodes.
// 2. Not proposer itself.
// 3. When `ISAACStateManager` starts, the node waits a ballot by proposer.
// 4. TimeoutINIT is a millisecond.
// 5. After milliseconds, the node broadcasts B(`SIGN`, `EXP`).
// 6. ISAACState is changed to `SIGN`.
// 7. TimeoutSIGN is a millisecond.
// 8. After timeout, the node broadcasts B(`ACCEPT`, `EXP`).
func TestStateSIGNTimeoutNotProposer(t *testing.T) {
	conf := consensus.NewISAACConfiguration()
	conf.TimeoutINIT = 200 * time.Millisecond
	conf.TimeoutSIGN = 200 * time.Millisecond
	conf.TimeoutACCEPT = time.Hour
	conf.TimeoutALLCONFIRM = time.Hour

	nodeRunners := createTestNodeRunner(3, conf)

	nr := nodeRunners[0]

	recv := make(chan struct{})
	b := NewTestBroadcaster(recv)
	nr.Consensus().SetBroadcaster(b)
	nr.Consensus().ConnectionManager().SetProposerCalculator(TheOtherProposerCalculator{
		nodeRunner: nr,
	})
	proposer := nr.Consensus().ConnectionManager().CalculateProposer(0, 0)

	require.NotEqual(t, nr.localNode.Address(), proposer)

	nr.Consensus().SetLatestConsensusedBlock(genesisBlock)

	nr.StartStateManager()
	defer nr.StopStateManager()

	<-recv
	require.Equal(t, 1, len(b.Messages()))

	<-recv
	require.Equal(t, 2, len(b.Messages()))

	init, sign, accept := 0, 0, 0
	for _, message := range b.Messages() {
		b, ok := message.(block.Ballot)
		require.True(t, ok)
		require.Equal(t, 0, len(b.Transactions()))
		switch b.State() {
		case ballot.StateINIT:
			init++
			require.Equal(t, ballot.VotingYES, b.Vote())
		case ballot.StateSIGN:
			sign++
			require.Equal(t, ballot.VotingEXP, b.Vote())
		case ballot.StateACCEPT:
			accept++
			require.Equal(t, ballot.VotingEXP, b.Vote())
		}
	}
	require.Equal(t, 0, init)
	require.Equal(t, 1, sign)
	require.Equal(t, 1, accept)
}

// 1. All 3 Nodes.
// 2. Proposer itself at round 0.
// 3. When `ISAACStateManager` starts, the node proposes a ballot.
// 4. ISAACState is changed to `SIGN`.
// 5. TimeoutSIGN is a millisecond.
// 6. After timeout, the node broadcasts B(`ACCEPT`, `EXP`).
func TestStateACCEPTTimeoutProposerThenNotProposer(t *testing.T) {
	conf := consensus.NewISAACConfiguration()
	conf.TimeoutINIT = time.Hour
	conf.TimeoutSIGN = 200 * time.Millisecond
	conf.TimeoutACCEPT = 200 * time.Millisecond
	conf.TimeoutALLCONFIRM = time.Hour

	nodeRunners := createTestNodeRunner(3, conf)

	nr := nodeRunners[0]

	recv := make(chan struct{})
	b := NewTestBroadcaster(recv)
	nr.Consensus().SetBroadcaster(b)
	nr.Consensus().ConnectionManager().SetProposerCalculator(&SelfProposerThenNotProposer{
		nodeRunner: nr,
	})

	proposer := nr.Consensus().ConnectionManager().CalculateProposer(0, 0)
	require.Equal(t, nr.localNode.Address(), proposer)

	proposer = nr.Consensus().ConnectionManager().CalculateProposer(0, 1)
	require.NotEqual(t, nr.localNode.Address(), proposer)

	nr.Consensus().SetLatestConsensusedBlock(genesisBlock)

	nr.StartStateManager()
	defer nr.StopStateManager()

	<-recv
	require.Equal(t, 1, len(b.Messages()))

	<-recv
	require.Equal(t, 2, len(b.Messages()))
	init, sign, accept := 0, 0, 0
	for _, message := range b.Messages() {
		b, ok := message.(block.Ballot)
		require.True(t, ok)
		require.Equal(t, 0, len(b.Transactions()))
		switch b.State() {
		case ballot.StateINIT:
			init++
			require.Equal(t, b.Vote(), ballot.VotingYES)
		case ballot.StateSIGN:
			sign++
		case ballot.StateACCEPT:
			accept++
			require.Equal(t, b.Vote(), ballot.VotingEXP)
		}
	}

	require.Equal(t, 1, init)
	require.Equal(t, 0, sign)
	require.Equal(t, 1, accept)
}

// 1. All 3 Nodes.
// 2. Not proposer itself.
// 3. When `ISAACStateManager` starts, the node waits for a proposed ballot.
// 4. TransitISAACState(SIGN) method is called.
// 5. ISAACState is changed to `SIGN`.
// 6. TimeoutSIGN is a millisecond.
// 7. After timeout, the node broadcasts B(`ACCEPT`, `EXP`).
func TestStateTransitFromTimeoutInitToAccept(t *testing.T) {
	conf := consensus.NewISAACConfiguration()
	conf.TimeoutINIT = time.Hour
	conf.TimeoutSIGN = 200 * time.Millisecond
	conf.TimeoutACCEPT = 200 * time.Millisecond
	conf.TimeoutALLCONFIRM = 200 * time.Millisecond

	nodeRunners := createTestNodeRunner(3, conf)
	nr := nodeRunners[0]

	recvTransit := make(chan struct{})
	nr.isaacStateManager.SetTransitSignal(func() {
		recvTransit <- struct{}{}
	})

	recvBroadcast := make(chan struct{})
	b := NewTestBroadcaster(recvBroadcast)
	nr.Consensus().SetBroadcaster(b)
	nr.Consensus().ConnectionManager().SetProposerCalculator(TheOtherProposerCalculator{
		nodeRunner: nr,
	})

	nr.Consensus().SetLatestConsensusedBlock(genesisBlock)

	nr.StartStateManager()
	defer nr.StopStateManager()
	<-recvTransit
	require.Equal(t, ballot.StateINIT, nr.isaacStateManager.State().BallotState)

	nr.TransitISAACState(nr.isaacStateManager.State().Round, ballot.StateSIGN)
	<-recvTransit
	require.Equal(t, ballot.StateSIGN, nr.isaacStateManager.State().BallotState)

	<-recvBroadcast
	require.Equal(t, 1, len(b.Messages()))

	for _, message := range b.Messages() {
		b, ok := message.(block.Ballot)
		require.True(t, ok)
		require.Equal(t, nr.localNode.Address(), b.Proposer())
		require.Equal(t, ballot.StateACCEPT, b.State())
		require.Equal(t, ballot.VotingEXP, b.Vote())
	}
}

// 1. All 3 Nodes.
// 1. Proposer itself.
// 1. When `ISAACStateManager` starts, the node proposes a ballot.
// 1. ISAACState is changed to `SIGN`.
// 1. TransitISAACState(ACCEPT) method is called.
// 1. ISAACState is changed to `ACCEPT`.
// 1. TimeoutACCEPT is a millisecond.
// 1. After timeout, ISAACState is back to `INIT`
func TestStateTransitFromTimeoutSignToAccept(t *testing.T) {
	conf := consensus.NewISAACConfiguration()
	conf.TimeoutINIT = time.Hour
	conf.TimeoutSIGN = time.Hour
	conf.TimeoutACCEPT = 200 * time.Millisecond
	conf.TimeoutALLCONFIRM = 200 * time.Millisecond

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
	for _, message := range b.Messages() {
		b, ok := message.(block.Ballot)
		require.True(t, ok)
		require.Equal(t, nr.localNode.Address(), b.Proposer())
		require.Equal(t, ballot.StateINIT, b.State())
		require.Equal(t, ballot.VotingYES, b.Vote())
	}

	nr.TransitISAACState(nr.isaacStateManager.State().Round, ballot.StateACCEPT)
	<-recv

	require.Equal(t, 2, len(b.Messages()))
	for _, message := range b.Messages() {
		b, ok := message.(block.Ballot)
		require.True(t, ok)
		require.Equal(t, nr.localNode.Address(), b.Proposer())
		require.Equal(t, ballot.StateINIT, b.State())
		require.Equal(t, ballot.VotingYES, b.Vote())
	}
}
