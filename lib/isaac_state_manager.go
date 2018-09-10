package sebak

import (
	"time"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/round"
)

// ISAACStateManager manages the ISAACState.
// The most important function `Start()` is called in StartStateManager() function in node_runner.go by goroutine.
type ISAACStateManager struct {
	nr            *NodeRunner
	state         ISAACState
	conf          *ISAACConfiguration
	stateTransit  chan ISAACState
	nextHeight    chan struct{}
	stop          chan struct{}
	transitSignal func() // `transitSignal` is function which is called when the ISAACState is changed.
}

func NewISAACStateManager(nr *NodeRunner) *ISAACStateManager {
	p := &ISAACStateManager{
		conf: NewISAACConfiguration(),
		nr:   nr,
	}
	p.stateTransit = make(chan ISAACState)
	p.nextHeight = make(chan struct{})
	p.stop = make(chan struct{})

	p.state = NewISAACState(
		round.Round{
			Number:      0,
			BlockHeight: 0,
		},
		common.BallotStateINIT,
	)

	p.transitSignal = func() {}

	return p
}

func (sm *ISAACStateManager) SetConf(conf *ISAACConfiguration) {
	sm.conf = conf
}

func (sm *ISAACStateManager) SetTransitSignal(f func()) {
	sm.transitSignal = f
}

func (sm *ISAACStateManager) TransitISAACState(round round.Round, ballotState common.BallotState) {
	current := sm.state
	target := NewISAACState(round, ballotState)

	if isTargetLater(current, target) {
		go func() {
			sm.stateTransit <- target
		}()
	}
}

func isTargetLater(state ISAACState, target ISAACState) (result bool) {
	if state.round.BlockHeight > target.round.BlockHeight {
		result = false
	} else if state.round.BlockHeight < target.round.BlockHeight {
		result = true
	} else { // state.round.BlockHeight == target.round.BlockHeight
		if state.round.Number > target.round.Number {
			result = false
		} else if state.round.Number < target.round.Number {
			result = true
		} else { // state.round.Number == target.round.Number
			if state.ballotState >= target.ballotState {
				result = false
			} else {
				result = true
			}
		}
	}
	return result
}

func (sm *ISAACStateManager) IncreaseRound() {
	sm.increaseRound()
}

func (sm *ISAACStateManager) increaseRound() {
	round := sm.state.round
	round.Number++
	sm.TransitISAACState(round, common.BallotStateINIT)
}

func (sm *ISAACStateManager) NextHeight() {
	go func() {
		sm.nextHeight <- struct{}{}
	}()
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
				if sm.state.ballotState == common.BallotStateACCEPT {
					sm.increaseRound()
					break
				}
				go sm.broadcastExpiredBallot(sm.state)
				sm.state.ballotState = sm.state.ballotState.Next()
				sm.resetTimer(timer, sm.state.ballotState)
				sm.transitSignal()

			case state := <-sm.stateTransit:
				switch state.ballotState {
				case common.BallotStateINIT:
					sm.proposeOrWait(timer, state)
				case common.BallotStateSIGN:
					sm.state = state
					sm.transitSignal()
					timer.Reset(sm.conf.TimeoutSIGN)
				case common.BallotStateACCEPT:
					sm.state = state
					sm.transitSignal()
					timer.Reset(sm.conf.TimeoutACCEPT)
				case common.BallotStateALLCONFIRM:
					sm.NextHeight()
				case common.BallotStateNONE:
					timer.Reset(sm.conf.TimeoutINIT)
					log.Error("Wrong ISAACState", "ISAACState", state)
				}

			case <-sm.nextHeight:
				round := sm.state.round
				round.BlockHeight++
				round.Number = 0
				sm.TransitISAACState(round, common.BallotStateINIT)

			case <-sm.stop:
				return
			}
		}
	}()
}

func (sm *ISAACStateManager) broadcastExpiredBallot(state ISAACState) {
	round := round.Round{
		Number:      state.round.Number,
		BlockHeight: sm.nr.consensus.LatestConfirmedBlock.Height,
		BlockHash:   sm.nr.consensus.LatestConfirmedBlock.Hash,
		TotalTxs:    sm.nr.consensus.LatestConfirmedBlock.TotalTxs,
	}

	newExpiredBallot := NewBallot(sm.nr.localNode, round, []string{})
	newExpiredBallot.SetVote(state.ballotState.Next(), common.VotingEXP)
	newExpiredBallot.Sign(sm.nr.localNode.Keypair(), sm.nr.networkID)

	sm.nr.ConnectionManager().Broadcast(*newExpiredBallot)
}

func (sm *ISAACStateManager) resetTimer(timer *time.Timer, state common.BallotState) {
	switch state {
	case common.BallotStateINIT:
		timer.Reset(sm.conf.TimeoutINIT)
	case common.BallotStateSIGN:
		timer.Reset(sm.conf.TimeoutSIGN)
	case common.BallotStateACCEPT:
		timer.Reset(sm.conf.TimeoutACCEPT)
	}
}

func (sm *ISAACStateManager) proposeOrWait(timer *time.Timer, state ISAACState) {
	timer.Reset(time.Duration(1 * time.Hour))
	proposer := sm.nr.CalculateProposer(state.round.BlockHeight, state.round.Number)
	log.Debug("calculated proposer", "proposer", proposer)

	if proposer == sm.nr.localNode.Address() {
		if err := sm.nr.proposeNewBallot(state.round.Number); err == nil {
			log.Debug("propose new ballot", "proposer", proposer, "round", state.round, "ballotState", common.BallotStateSIGN)
			sm.state = state
			sm.state.ballotState = common.BallotStateSIGN
			timer.Reset(sm.conf.TimeoutSIGN)
			sm.transitSignal()
		} else {
			log.Error("failed to proposeNewBallot", "height", sm.nr.consensus.LatestConfirmedBlock.Height, "error", err)
			sm.state = state
			timer.Reset(sm.conf.TimeoutINIT)
		}
	} else {
		sm.state = state
		timer.Reset(sm.conf.TimeoutINIT)
		sm.transitSignal()
	}
}

func (sm *ISAACStateManager) State() ISAACState {
	return sm.state
}

func (sm *ISAACStateManager) Stop() {
	go func() {
		sm.stop <- struct{}{}
	}()
}
