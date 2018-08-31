package sebak

import (
	"time"

	"boscoin.io/sebak/lib/common"
)

type NodeRunnerStateManager struct {
	nr            *NodeRunner
	state         NodeRunnerState
	conf          *NodeRunnerConfiguration
	stateTransit  chan NodeRunnerState
	increaseRound chan bool
	resetRound    chan bool
	stop          chan bool
	on            bool
}

func NewNodeRunnerStateManager(nr *NodeRunner) *NodeRunnerStateManager {
	p := &NodeRunnerStateManager{
		conf: NewNodeRunnerConfiguration(),
		nr:   nr,
	}
	p.stateTransit = make(chan NodeRunnerState)
	p.increaseRound = make(chan bool)
	p.resetRound = make(chan bool)
	p.stop = make(chan bool)
	p.on = false

	return p
}

func (sm *NodeRunnerStateManager) SetConf(conf *NodeRunnerConfiguration) {
	sm.conf = conf
}

func (sm *NodeRunnerStateManager) TransitNodeRunnerState(round Round, ballotState sebakcommon.BallotState) {
	currentState := sm.state
	targetState := NewNodeRunnerState(round, ballotState)

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
		sm.stateTransit <- NewNodeRunnerState(round, ballotState)
	}()
}

func (sm *NodeRunnerStateManager) IncreaseRound() {
	go func() {
		sm.increaseRound <- true
	}()
}

func (sm *NodeRunnerStateManager) ResetRound() {
	go func() {
		sm.resetRound <- true
	}()
}

func (sm *NodeRunnerStateManager) Start() {
	oneHour := time.Duration(1 * time.Hour)
	if sm.on {
		return
	}
	sm.on = true
	timer := time.NewTimer(sm.conf.GetTimeout(sebakcommon.BallotStateINIT))
	sm.state = NewNodeRunnerState(
		Round{
			Number:      0,
			BlockHeight: 0,
		},
		sebakcommon.BallotStateNONE,
	)

	for {
		select {
		case <-timer.C:
			if sm.state.ballotState == sebakcommon.BallotStateACCEPT {
				sm.IncreaseRound()
				break
			}
			go sm.broadcastExpiredBallot(sm.state)
			sm.TransitNodeRunnerState(sm.state.round, sm.state.ballotState.Next())
		case sm.state = <-sm.stateTransit:
			switch sm.state.ballotState {
			case sebakcommon.BallotStateINIT:
				timer.Reset(oneHour)
				proposer := sm.nr.CalculateProposer(sm.state.round.BlockHeight, sm.state.round.Number)
				log.Debug("calculated proposer", "proposer", proposer)

				if proposer == sm.nr.localNode.Address() {
					if err := sm.nr.proposeNewBallot(sm.state.round.Number); err == nil {
						log.Debug("propose new ballot", "proposer", proposer, "round", sm.state.round, "ballotState", sebakcommon.BallotStateSIGN)
						sm.TransitNodeRunnerState(sm.state.round, sebakcommon.BallotStateSIGN)
					} else {
						sm.nr.log.Error("failed to proposeNewBallot", "height", sm.nr.consensus.LatestConfirmedBlock.Height, "error", err)
						sm.TransitNodeRunnerState(sm.state.round, sebakcommon.BallotStateINIT)
					}
				} else {
					timer.Reset(sm.conf.TimeoutINIT)
				}
			case sebakcommon.BallotStateSIGN:
				timer.Reset(sm.conf.TimeoutSIGN)
			case sebakcommon.BallotStateACCEPT:
				timer.Reset(sm.conf.TimeoutACCEPT)
			case sebakcommon.BallotStateALLCONFIRM:
				sm.ResetRound()
			case sebakcommon.BallotStateNONE:
				timer.Reset(sm.conf.TimeoutINIT)
			}
		case <-sm.increaseRound:
			round := sm.state.round
			round.Number++
			sm.TransitNodeRunnerState(round, sebakcommon.BallotStateINIT)
		case <-sm.resetRound:
			round := sm.state.round
			round.BlockHeight++
			round.Number = 0
			sm.TransitNodeRunnerState(round, sebakcommon.BallotStateINIT)
		case <-sm.stop:
			sm.on = false
			return
		}
	}
}

func (sm *NodeRunnerStateManager) broadcastExpiredBallot(state NodeRunnerState) {
	round := Round{
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

func (sm *NodeRunnerStateManager) State() NodeRunnerState {
	return sm.state
}

func (sm *NodeRunnerStateManager) Stop() {
	go func() {
		sm.stop <- true
	}()
}
