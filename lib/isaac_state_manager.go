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
	conf         *IsaacConfiguration
	stateTransit chan IsaacState
	resetRound   chan bool
	stop         chan bool
}

func NewIsaacStateManager(nr *NodeRunner) *IsaacStateManager {
	p := &IsaacStateManager{
		conf: NewIsaacConfiguration(),
		nr:   nr,
	}
	p.stateTransit = make(chan IsaacState)
	p.resetRound = make(chan bool)
	p.stop = make(chan bool)

	return p
}

func (sm *IsaacStateManager) SetConf(conf *IsaacConfiguration) {
	sm.conf = conf
}

func (sm *IsaacStateManager) TransitIsaacState(round round.Round, ballotState sebakcommon.BallotState) {
	currentState := sm.state
	targetState := NewIsaacState(round, ballotState)

	if currentState.round.BlockHeight > targetState.round.BlockHeight {
		return
	} else if currentState.round.BlockHeight < targetState.round.BlockHeight {
		// transit
	} else { // currentState.round.BlockHeight == targetState.round.BlockHeight
		if currentState.round.Number > round.Number {
			return
		} else if currentState.round.Number < targetState.round.Number {
			// transit
		} else { // currentState.round.Number == targetState.round.Number
			if currentState.ballotState >= ballotState {
				return
			} else {
				// transit
			}
		}
	}

	go func() {
		sm.stateTransit <- NewIsaacState(round, ballotState)
	}()
}

func (sm *IsaacStateManager) increaseRound() {
	round := sm.state.round
	round.Number++
	sm.TransitIsaacState(round, sebakcommon.BallotStateINIT)
}

func (sm *IsaacStateManager) ResetRound() {
	go func() {
		sm.resetRound <- true
	}()
}

// In `Start()` method a node proposes ballot.
// Or it sets or resets timeout. If it is expired, it broadcasts B(`EXP`).
// And it manages the node round.
func (sm *IsaacStateManager) Start() {
	oneHour := time.Duration(1 * time.Hour)
	timer := time.NewTimer(sm.conf.GetTimeout(sebakcommon.BallotStateINIT))
	sm.state = NewIsaacState(
		round.Round{
			Number:      0,
			BlockHeight: 0,
		},
		sebakcommon.BallotStateNONE,
	)

	for {
		select {
		case <-timer.C:
			if sm.state.ballotState == sebakcommon.BallotStateACCEPT {
				sm.increaseRound()
				break
			}
			go sm.broadcastExpiredBallot(sm.state)
			state := sm.state
			state.ballotState = sm.state.ballotState.Next()
			sm.transitState(timer, state)
		case state := <-sm.stateTransit:
			switch state.ballotState {
			case sebakcommon.BallotStateINIT:
				proposer := sm.nr.CalculateProposer(state.round.BlockHeight, state.round.Number)
				log.Debug("calculated proposer", "proposer", proposer)

				if proposer == sm.nr.localNode.Address() {
					timer.Reset(oneHour)
					if err := sm.nr.proposeNewBallot(state.round.Number); err == nil {
						log.Debug("propose new ballot", "proposer", proposer, "round", state.round, "ballotState", sebakcommon.BallotStateSIGN)
						state.ballotState = sebakcommon.BallotStateSIGN
						sm.transitState(timer, state)
					} else {
						sm.nr.log.Error("failed to proposeNewBallot", "height", sm.nr.consensus.LatestConfirmedBlock.Height, "error", err)
						state.ballotState = sebakcommon.BallotStateINIT
						sm.transitState(timer, state)
					}
				} else {
					state.ballotState = sebakcommon.BallotStateINIT
					sm.transitState(timer, state)
				}
			case sebakcommon.BallotStateSIGN:
				sm.transitState(timer, state)
			case sebakcommon.BallotStateACCEPT:
				sm.transitState(timer, state)
			case sebakcommon.BallotStateALLCONFIRM:
				sm.ResetRound()
			case sebakcommon.BallotStateNONE:
				sm.transitState(timer, state)
			}
		case <-sm.resetRound:
			round := sm.state.round
			round.BlockHeight++
			round.Number = 0
			sm.TransitIsaacState(round, sebakcommon.BallotStateINIT)
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
	newExpiredBallot.SetVote(state.ballotState.Next(), sebakcommon.VotingEXP)
	newExpiredBallot.Sign(sm.nr.localNode.Keypair(), sm.nr.networkID)

	sm.nr.ConnectionManager().Broadcast(*newExpiredBallot)
}

func (sm *IsaacStateManager) transitState(timer *time.Timer, state IsaacState) {
	switch state.ballotState {
	case sebakcommon.BallotStateINIT:
		timer.Reset(sm.conf.TimeoutINIT)
	case sebakcommon.BallotStateSIGN:
		timer.Reset(sm.conf.TimeoutSIGN)
	case sebakcommon.BallotStateACCEPT:
		timer.Reset(sm.conf.TimeoutACCEPT)
	}
	sm.state = state
}

func (sm *IsaacStateManager) State() IsaacState {
	return sm.state
}

func (sm *IsaacStateManager) Stop() {
	go func() {
		sm.stop <- true
	}()
}
