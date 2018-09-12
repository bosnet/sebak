package runner

import (
	"sync"
	"time"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/consensus"
	"boscoin.io/sebak/lib/consensus/round"
)

// ISAACStateManager manages the ISAACState.
// The most important function `Start()` is called in StartStateManager() function in node_runner.go by goroutine.
type ISAACStateManager struct {
	sync.RWMutex

	nr            *NodeRunner
	state         consensus.ISAACState
	Conf          *consensus.ISAACConfiguration
	stateTransit  chan consensus.ISAACState
	stop          chan struct{}
	transitSignal func() // `transitSignal` is function which is called when the ISAACState is changed.
}

func NewISAACStateManager(nr *NodeRunner, conf *consensus.ISAACConfiguration) *ISAACStateManager {
	p := &ISAACStateManager{
		Conf: consensus.NewISAACConfiguration(),
		nr:   nr,
	}
	p.stateTransit = make(chan consensus.ISAACState)
	p.stop = make(chan struct{})

	p.state = consensus.ISAACState{
		Round: round.Round{
			Number:      0,
			BlockHeight: 0,
		},
		BallotState: common.BallotStateINIT,
	}

	p.transitSignal = func() {}

	return p
}

func (sm *ISAACStateManager) SetTransitSignal(f func()) {
	sm.transitSignal = f
}

func (sm *ISAACStateManager) TransitISAACState(round round.Round, ballotState common.BallotState) {
	sm.RLock()
	current := sm.state
	sm.RUnlock()

	target := consensus.ISAACState{
		Round:       round,
		BallotState: ballotState,
	}

	if current.IsLater(target) {
		go func() {
			sm.stateTransit <- target
		}()
	}
}

func (sm *ISAACStateManager) IncreaseRound() {
	round := sm.State().Round
	round.Number++
	sm.TransitISAACState(round, common.BallotStateINIT)
}

func (sm *ISAACStateManager) NextHeight() {
	round := sm.State().Round
	round.BlockHeight++
	round.Number = 0
	sm.TransitISAACState(round, common.BallotStateINIT)
}

// In `Start()` method a node proposes ballot.
// Or it sets or resets timeout. If it is expired, it broadcasts B(`EXP`).
// And it manages the node round.
func (sm *ISAACStateManager) Start() {
	go func() {
		timer := time.NewTimer(time.Duration(1 * time.Hour))
		for {
			select {
			case <-timer.C:
				if sm.State().BallotState == common.BallotStateACCEPT {
					sm.IncreaseRound()
					break
				}
				go sm.broadcastExpiredBallot(sm.State())
				sm.setBallotState(sm.State().BallotState.Next())
				sm.resetTimer(timer, sm.State().BallotState)
				sm.transitSignal()

			case state := <-sm.stateTransit:
				switch state.BallotState {
				case common.BallotStateINIT:
					sm.proposeOrWait(timer, state)
				case common.BallotStateSIGN:
					sm.setState(state)
					sm.transitSignal()
					timer.Reset(sm.Conf.TimeoutSIGN)
				case common.BallotStateACCEPT:
					sm.setState(state)
					sm.transitSignal()
					timer.Reset(sm.Conf.TimeoutACCEPT)
				case common.BallotStateALLCONFIRM:
					sm.NextHeight()
				case common.BallotStateNONE:
					timer.Reset(sm.Conf.TimeoutINIT)
					log.Error("Wrong ISAACState", "ISAACState", state)
				}

			case <-sm.stop:
				return
			}
		}
	}()
}

func (sm *ISAACStateManager) broadcastExpiredBallot(state consensus.ISAACState) {
	round := round.Round{
		Number:      state.Round.Number,
		BlockHeight: sm.nr.consensus.LatestConfirmedBlock.Height,
		BlockHash:   sm.nr.consensus.LatestConfirmedBlock.Hash,
		TotalTxs:    sm.nr.consensus.LatestConfirmedBlock.TotalTxs,
	}

	newExpiredBallot := block.NewBallot(sm.nr.localNode, round, []string{})
	newExpiredBallot.SetVote(state.BallotState.Next(), common.VotingEXP)
	newExpiredBallot.Sign(sm.nr.localNode.Keypair(), sm.nr.networkID)

	sm.nr.ConnectionManager().Broadcast(*newExpiredBallot)
}

func (sm *ISAACStateManager) resetTimer(timer *time.Timer, state common.BallotState) {
	switch state {
	case common.BallotStateINIT:
		timer.Reset(sm.Conf.TimeoutINIT)
	case common.BallotStateSIGN:
		timer.Reset(sm.Conf.TimeoutSIGN)
	case common.BallotStateACCEPT:
		timer.Reset(sm.Conf.TimeoutACCEPT)
	}
}

// In proposeOrWait,
// if nr.localNode is proposer, it proposes new ballot,
// but if not, it waits for receiving ballot from the other proposer.
func (sm *ISAACStateManager) proposeOrWait(timer *time.Timer, state consensus.ISAACState) {
	timer.Reset(time.Duration(1 * time.Hour))
	proposer := sm.nr.CalculateProposer(state.Round.BlockHeight, state.Round.Number)
	log.Debug("calculated proposer", "proposer", proposer)

	if proposer == sm.nr.localNode.Address() {
		if err := sm.nr.proposeNewBallot(state.Round.Number); err == nil {
			log.Debug("propose new ballot", "proposer", proposer, "round", state.Round, "ballotState", common.BallotStateSIGN)
			state.BallotState = common.BallotStateSIGN
			sm.setState(state)

			timer.Reset(sm.Conf.TimeoutSIGN)
			sm.transitSignal()
		} else {
			log.Error("failed to proposeNewBallot", "height", sm.nr.consensus.LatestConfirmedBlock.Height, "error", err)
			sm.setState(state)
			timer.Reset(sm.Conf.TimeoutINIT)
		}
	} else {
		sm.setState(state)
		timer.Reset(sm.Conf.TimeoutINIT)
		sm.transitSignal()
	}
}

func (sm *ISAACStateManager) State() consensus.ISAACState {
	sm.RLock()
	defer sm.RUnlock()
	return sm.state
}

func (sm *ISAACStateManager) setState(state consensus.ISAACState) {
	sm.Lock()
	defer sm.Unlock()
	sm.state = state

	return
}

func (sm *ISAACStateManager) setBallotState(ballotState common.BallotState) {
	sm.Lock()
	defer sm.Unlock()
	sm.state.BallotState = ballotState

	return
}

func (sm *ISAACStateManager) Stop() {
	go func() {
		sm.stop <- struct{}{}
	}()
}
