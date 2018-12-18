// We can test that a node broadcast propose ballot or B(`EXP`) in ISAACStateManager.
package runner

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"boscoin.io/sebak/lib/ballot"
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/consensus"
	"boscoin.io/sebak/lib/voting"
)

// 1. All 3 Nodes.
// 2. Proposer itself.
// 3. When `ISAACStateManager` starts, the node proposes ballot to validators.
func TestStateINITProposer(t *testing.T) {
	conf := common.NewTestConfig()
	conf.TimeoutINIT = time.Hour
	conf.TimeoutSIGN = time.Hour
	conf.TimeoutACCEPT = time.Hour

	recv := make(chan struct{})
	nr, _, cm := createNodeRunnerForTesting(3, conf, recv)

	nr.startStateManager()
	defer nr.StopStateManager()

	<-recv
	require.Equal(t, 1, len(cm.Messages()))
	for _, message := range cm.Messages() {
		// This message must be proposed ballot
		b, ok := message.(ballot.Ballot)
		require.True(t, ok)
		require.Equal(t, nr.localNode.Address(), b.Proposer())
	}
}

// 1. All 3 Nodes.
// 2. Not proposer itself.
// 3. When `ISAACStateManager` starts, the node waits a ballot by proposer.
// 4. But TimeoutINIT is an hour, so it doesn't broadcast anything.
func TestStateINITNotProposer(t *testing.T) {
	conf := common.NewTestConfig()
	conf.TimeoutINIT = time.Hour
	conf.TimeoutSIGN = time.Hour
	conf.TimeoutACCEPT = time.Hour

	recv := make(chan struct{})
	nr, _, _ := createNodeRunnerForTesting(3, conf, recv)
	nr.Consensus().SetProposerSelector(OtherSelector{cm: nr.ConnectionManager(), localNode: nr.Node()})

	cm, ok := nr.Consensus().ConnectionManager().(*TestConnectionManager)
	require.True(t, ok)

	nr.startStateManager()
	defer nr.StopStateManager()
	time.Sleep(1 * time.Second)

	require.Equal(t, 0, len(cm.Messages()))
}

// 1. All 3 Nodes.
// 2. Not proposer itself.
// 3. When `ISAACStateManager` starts, the node waits a ballot by proposer.
// 4. But TimeoutINIT is a millisecond.
// 5. After timeout, ISAACState is changed to `SIGN`.
func TestStateINITTimeoutNotProposer(t *testing.T) {
	conf := common.NewTestConfig()
	conf.TimeoutINIT = 200 * time.Millisecond
	conf.TimeoutSIGN = time.Hour
	conf.TimeoutACCEPT = time.Hour

	nr, _, _ := createNodeRunnerForTesting(3, conf, nil)
	nr.Consensus().SetProposerSelector(OtherSelector{cm: nr.ConnectionManager(), localNode: nr.Node()})

	_, ok := nr.Consensus().ConnectionManager().(*TestConnectionManager)
	require.True(t, ok)

	proposer := nr.Consensus().SelectProposer(0, 0)

	require.NotEqual(t, nr.localNode.Address(), proposer)

	recv := make(chan consensus.ISAACState)
	nr.isaacStateManager.SetTransitSignal(func(state consensus.ISAACState) {
		recv <- state
	})

	nr.startStateManager()
	defer nr.StopStateManager()
	state := <-recv
	require.Equal(t, ballot.StateINIT, state.BallotState)
	state = <-recv
	require.Equal(t, ballot.StateSIGN, state.BallotState)
}

// 1. All 3 Nodes.
// 2. Proposer itself.
// 3. When `ISAACStateManager` starts, the node proposes B(`INIT`, `YES`) to validators.
// 4. Then ISAACState will be changed to `SIGN`.
// 5. But TimeoutSIGN is a millisecond.
// 6. After timeout, the node broadcasts B(`SIGN`, `EXP`)
func TestStateSIGNTimeoutProposer(t *testing.T) {
	conf := common.NewTestConfig()
	conf.TimeoutINIT = time.Hour
	conf.TimeoutSIGN = 200 * time.Millisecond
	conf.TimeoutACCEPT = time.Hour

	recv := make(chan struct{})
	nr, _, cm := createNodeRunnerForTesting(3, conf, recv)

	proposer := nr.Consensus().SelectProposer(0, 0)

	require.Equal(t, nr.localNode.Address(), proposer)

	nr.startStateManager()
	defer nr.StopStateManager()

	require.Equal(t, ballot.StateINIT, nr.isaacStateManager.State().BallotState)

	<-recv
	require.Equal(t, 1, len(cm.Messages()))
	require.Equal(t, ballot.StateINIT, nr.isaacStateManager.State().BallotState)

	state := nr.isaacStateManager.State()
	nr.isaacStateManager.TransitISAACState(state.Height, state.Round, ballot.StateSIGN)

	<-recv
	require.Equal(t, ballot.StateSIGN, nr.isaacStateManager.State().BallotState)
	require.Equal(t, 2, len(cm.Messages()))

	init, sign, accept := 0, 0, 0
	for _, message := range cm.Messages() {
		b, ok := message.(ballot.Ballot)
		require.True(t, ok)
		require.Equal(t, 0, len(b.Transactions()))
		switch b.State() {
		case ballot.StateINIT:
			init++
			require.Equal(t, voting.YES, b.Vote())
		case ballot.StateSIGN:
			sign++
			require.Equal(t, voting.EXP, b.Vote())
		case ballot.StateACCEPT:
			accept++
			require.Equal(t, voting.EXP, b.Vote())
		}
	}
	require.Equal(t, 1, init)
	require.Equal(t, 1, sign)
	require.Equal(t, 0, accept)
}

// 1. All 3 Nodes.
// 2. Not proposer itself.
// 3. When `ISAACStateManager` starts, the node waits a ballot by proposer.
// 4. TimeoutINIT is a millisecond.
// 5. ISAACState is changed to `SIGN`.
// 6. TimeoutSIGN is a millisecond.
// 7. After timeout, the node broadcasts B(`SIGN`, `EXP`).
func TestStateSIGNTimeoutNotProposer(t *testing.T) {
	conf := common.NewTestConfig()
	conf.TimeoutINIT = 200 * time.Millisecond
	conf.TimeoutSIGN = 200 * time.Millisecond
	conf.TimeoutACCEPT = time.Hour

	recv := make(chan struct{})
	nr, _, _ := createNodeRunnerForTesting(3, conf, recv)
	nr.Consensus().SetProposerSelector(OtherSelector{cm: nr.ConnectionManager(), localNode: nr.Node()})

	cm, ok := nr.Consensus().ConnectionManager().(*TestConnectionManager)
	require.True(t, ok)

	proposer := nr.Consensus().SelectProposer(0, 0)

	require.NotEqual(t, nr.localNode.Address(), proposer)

	nr.startStateManager()
	defer nr.StopStateManager()

	recvTransit := make(chan consensus.ISAACState)
	nr.isaacStateManager.SetTransitSignal(func(state consensus.ISAACState) {
		recvTransit <- state
	})

	state := <-recvTransit
	require.Equal(t, ballot.StateINIT, state.BallotState)

	state = <-recvTransit
	require.Equal(t, ballot.StateSIGN, state.BallotState)

	<-recv
	require.Equal(t, 1, len(cm.Messages()))

	init, sign, accept := 0, 0, 0
	for _, message := range cm.Messages() {
		b, ok := message.(ballot.Ballot)
		require.True(t, ok)
		require.Equal(t, 0, len(b.Transactions()))
		switch b.State() {
		case ballot.StateINIT:
			init++
			require.Equal(t, voting.YES, b.Vote())
		case ballot.StateSIGN:
			sign++
			require.Equal(t, voting.EXP, b.Vote())
		case ballot.StateACCEPT:
			accept++
			require.Equal(t, voting.EXP, b.Vote())
		}
	}
	require.Equal(t, 0, init)
	require.Equal(t, 1, sign)
	require.Equal(t, 0, accept)
}

// 1. All 3 Nodes.
// 2. Proposer itself at round 0.
// 3. When `ISAACStateManager` starts, the node proposes a ballot.
// 4. ISAACState is changed to `SIGN`.
// 5. TimeoutSIGN is a millisecond.
// 6. After timeout, the node broadcasts B(`SIGN`, `EXP`).
func TestStateACCEPTTimeoutProposerThenNotProposer(t *testing.T) {
	conf := common.NewTestConfig()
	conf.TimeoutINIT = time.Hour
	conf.TimeoutSIGN = 200 * time.Millisecond
	conf.TimeoutACCEPT = 200 * time.Millisecond

	recv := make(chan struct{})
	nr, _, _ := createNodeRunnerForTesting(3, conf, recv)
	nr.Consensus().SetProposerSelector(SelfThenOtherSelector{cm: nr.ConnectionManager(), localNode: nr.Node()})

	cm, ok := nr.Consensus().ConnectionManager().(*TestConnectionManager)
	require.True(t, ok)

	proposer := nr.Consensus().SelectProposer(0, 0)
	require.Equal(t, nr.localNode.Address(), proposer)

	proposer = nr.Consensus().SelectProposer(0, 1)
	require.NotEqual(t, nr.localNode.Address(), proposer)

	nr.startStateManager()
	defer nr.StopStateManager()

	<-recv
	require.Equal(t, 1, len(cm.Messages()))

	state := nr.isaacStateManager.State()
	nr.isaacStateManager.TransitISAACState(state.Height, state.Round, ballot.StateSIGN)

	<-recv
	require.Equal(t, 2, len(cm.Messages()))
	init, sign, accept := 0, 0, 0
	for _, message := range cm.Messages() {
		b, ok := message.(ballot.Ballot)
		require.True(t, ok)
		require.Equal(t, 0, len(b.Transactions()))
		switch b.State() {
		case ballot.StateINIT:
			init++
			require.Equal(t, b.Vote(), voting.YES)
		case ballot.StateSIGN:
			sign++
		case ballot.StateACCEPT:
			accept++
			require.Equal(t, b.Vote(), voting.EXP)
		}
	}

	require.Equal(t, 1, init)
	require.Equal(t, 1, sign)
	require.Equal(t, 0, accept)
}

// 1. All 3 Nodes.
// 2. Not proposer itself.
// 3. When `ISAACStateManager` starts, the node waits for a proposed ballot.
// 4. TransitISAACState(SIGN) method is called.
// 5. ISAACState is changed to `SIGN`.
// 6. TimeoutSIGN is a millisecond.
// 7. After timeout, the node broadcasts B(`SIGN`, `EXP`).
func TestStateTransitFromTimeoutInitToAccept(t *testing.T) {
	conf := common.NewTestConfig()
	conf.TimeoutINIT = time.Hour
	conf.TimeoutSIGN = 200 * time.Millisecond
	conf.TimeoutACCEPT = 200 * time.Millisecond

	recvBroadcast := make(chan struct{})
	nr, _, _ := createNodeRunnerForTesting(3, conf, recvBroadcast)
	nr.Consensus().SetProposerSelector(OtherSelector{cm: nr.ConnectionManager(), localNode: nr.Node()})

	recvTransit := make(chan consensus.ISAACState)
	nr.isaacStateManager.SetTransitSignal(func(state consensus.ISAACState) {
		recvTransit <- state
	})

	cm, ok := nr.Consensus().ConnectionManager().(*TestConnectionManager)
	require.True(t, ok)

	nr.startStateManager()
	defer nr.StopStateManager()
	<-recvTransit
	state := nr.isaacStateManager.State()
	require.Equal(t, ballot.StateINIT, state.BallotState)

	basis := voting.Basis{
		Height: state.Height,
		Round:  state.Round,
	}

	nr.TransitISAACState(basis, ballot.StateSIGN)
	<-recvTransit
	require.Equal(t, ballot.StateSIGN, nr.isaacStateManager.State().BallotState)

	<-recvBroadcast
	require.Equal(t, 1, len(cm.Messages()))

	for _, message := range cm.Messages() {
		b, ok := message.(ballot.Ballot)
		require.True(t, ok)
		require.NotEqual(t, nr.localNode.Address(), b.Proposer())
		require.Equal(t, ballot.StateSIGN, b.State())
		require.Equal(t, voting.EXP, b.Vote())
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
	conf := common.NewTestConfig()
	conf.TimeoutINIT = time.Hour
	conf.TimeoutSIGN = time.Hour
	conf.TimeoutACCEPT = 200 * time.Millisecond

	recv := make(chan struct{})
	nr, _, cm := createNodeRunnerForTesting(3, conf, recv)

	nr.startStateManager()
	defer nr.StopStateManager()
	<-recv

	require.Equal(t, 1, len(cm.Messages()))
	for _, message := range cm.Messages() {
		b, ok := message.(ballot.Ballot)
		require.True(t, ok)
		require.Equal(t, nr.localNode.Address(), b.Proposer())
		require.Equal(t, ballot.StateINIT, b.State())
		require.Equal(t, voting.YES, b.Vote())
	}

	state := nr.isaacStateManager.State()

	basis := voting.Basis{
		Height: state.Height,
		Round:  state.Round,
	}

	nr.TransitISAACState(basis, ballot.StateACCEPT)
	<-recv

	require.Equal(t, 2, len(cm.Messages()))
	init, sign, accept := 0, 0, 0
	for _, message := range cm.Messages() {
		b, ok := message.(ballot.Ballot)
		require.True(t, ok)
		require.Equal(t, nr.localNode.Address(), b.Proposer())
		switch b.State() {
		case ballot.StateINIT:
			require.Equal(t, voting.YES, b.Vote())
			init++
		case ballot.StateSIGN:
			sign++
		case ballot.StateACCEPT:
			require.Equal(t, voting.EXP, b.Vote())
			accept++
		}
	}

	require.Equal(t, 1, init)
	require.Equal(t, 0, sign)
	require.Equal(t, 1, accept)
}
