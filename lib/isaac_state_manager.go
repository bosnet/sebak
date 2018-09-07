// IsaacStateManager manages the IsaacState.
// The most important function `Start()` is called in StartStateManager() function in node_runner.go by goroutine.

package sebak

import (
	"time"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/round"
)

type IsaacStateManager struct {
	nr           *NodeRunner
	state        IsaacState
	conf         *ISAACConfiguration
	stateTransit chan IsaacState
	nextHeight   chan bool
	stop         chan bool
}

func NewIsaacStateManager(nr *NodeRunner) *IsaacStateManager {
	p := &IsaacStateManager{
		conf: NewISAACConfiguration(),
		nr:   nr,
	}
	p.stateTransit = make(chan IsaacState)
	p.nextHeight = make(chan bool)
	p.stop = make(chan bool)

	return p
}

func (sm *IsaacStateManager) SetConf(conf *ISAACConfiguration) {
	sm.conf = conf
}

func (sm *IsaacStateManager) TransitIsaacState(round round.Round, ballotState common.BallotState) {
	current := sm.state
	target := NewIsaacState(round, ballotState)

	if isTargetLater(current, target) {
		go func() {
			sm.stateTransit <- target
		}()
	}
}

func isTargetLater(current IsaacState, target IsaacState) (result bool) {
	if current.round.BlockHeight > target.round.BlockHeight {
		result = false
	} else if current.round.BlockHeight < target.round.BlockHeight {
		result = true
	} else { // current.round.BlockHeight == target.round.BlockHeight
		if current.round.Number > target.round.Number {
			result = false
		} else if current.round.Number < target.round.Number {
			result = true
		} else { // current.round.Number == target.round.Number
			if current.ballotState >= target.ballotState {
				result = false
			} else {
				result = true
			}
		}
	}
	return result
}

func (sm *IsaacStateManager) IncreaseRound() {
	sm.increaseRound()
}

func (sm *IsaacStateManager) increaseRound() {
	round := sm.state.round
	round.Number++
	sm.TransitIsaacState(round, common.BallotStateINIT)
}

func (sm *IsaacStateManager) NextHeight() {
	go func() {
		sm.nextHeight <- true
	}()
}

// In `Start()` method a node proposes ballot.
// Or it sets or resets timeout. If it is expired, it broadcasts B(`EXP`).
// And it manages the node round.
func (sm *IsaacStateManager) Start() {
	timer := time.NewTimer(time.Duration(1 * time.Hour))
	sm.state = NewIsaacState(
		round.Round{
			Number:      0,
			BlockHeight: 0,
		},
		common.BallotStateINIT,
	)

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

		case state := <-sm.stateTransit:
			switch state.ballotState {
			case common.BallotStateINIT:
				sm.proposeOrWait(timer, state)
			case common.BallotStateSIGN:
				sm.state = state
				timer.Reset(sm.conf.TimeoutSIGN)
			case common.BallotStateACCEPT:
				sm.state = state
				timer.Reset(sm.conf.TimeoutACCEPT)
			case common.BallotStateALLCONFIRM:
				sm.NextHeight()
			case common.BallotStateNONE:
				timer.Reset(sm.conf.TimeoutINIT)
				log.Error("Wrong IsaacState", "IsaacState", state)
			}

		case <-sm.nextHeight:
			round := sm.state.round
			round.BlockHeight++
			round.Number = 0
			sm.TransitIsaacState(round, common.BallotStateINIT)

		case <-sm.stop:
			return
		}
	}
}

func (sm *IsaacStateManager) broadcastExpiredBallot(state IsaacState) {
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

func (sm *IsaacStateManager) resetTimer(timer *time.Timer, state common.BallotState) {
	switch state {
	case common.BallotStateINIT:
		timer.Reset(sm.conf.TimeoutINIT)
	case common.BallotStateSIGN:
		timer.Reset(sm.conf.TimeoutSIGN)
	case common.BallotStateACCEPT:
		timer.Reset(sm.conf.TimeoutACCEPT)
	}
}

func (sm *IsaacStateManager) proposeOrWait(timer *time.Timer, state IsaacState) {
	timer.Reset(time.Duration(1 * time.Hour))
	proposer := sm.nr.CalculateProposer(state.round.BlockHeight, state.round.Number)
	log.Debug("calculated proposer", "proposer", proposer)

	if proposer == sm.nr.localNode.Address() {
		if err := sm.nr.proposeNewBallot(state.round.Number); err == nil {
			log.Debug("propose new ballot", "proposer", proposer, "round", state.round, "ballotState", common.BallotStateSIGN)
			sm.state = state
			sm.state.ballotState = common.BallotStateSIGN
			timer.Reset(sm.conf.TimeoutSIGN)
		} else {
			log.Error("failed to proposeNewBallot", "height", sm.nr.consensus.LatestConfirmedBlock.Height, "error", err)
			sm.state = state
			timer.Reset(sm.conf.TimeoutINIT)
		}
	} else {
		sm.state = state
		timer.Reset(sm.conf.TimeoutINIT)
	}
}

func (sm *IsaacStateManager) State() IsaacState {
	return sm.state
}

func (sm *IsaacStateManager) Stop() {
	go func() {
		sm.stop <- true
	}()
}
