package sebak

import (
	"github.com/spikeekips/sebak/lib/common"
	"github.com/spikeekips/sebak/lib/error"
)

type ISAAC struct {
	sebakcommon.SafeLock

	networkID             []byte
	Node                  sebakcommon.Node
	VotingThresholdPolicy sebakcommon.VotingThresholdPolicy

	Boxes *BallotBoxes
}

func NewISAAC(networkID []byte, node sebakcommon.Node, votingThresholdPolicy sebakcommon.VotingThresholdPolicy) (is *ISAAC, err error) {
	is = &ISAAC{
		networkID: networkID,
		Node:      node,
		VotingThresholdPolicy: votingThresholdPolicy,
		Boxes: NewBallotBoxes(),
	}

	return
}

func (is *ISAAC) NetworkID() []byte {
	return is.networkID
}

func (is *ISAAC) GetNode() sebakcommon.Node {
	return is.Node
}

func (is *ISAAC) HasMessage(message sebakcommon.Message) bool {
	return is.Boxes.HasMessage(message)
}

func (is *ISAAC) HasMessageByHash(h string) bool {
	return is.Boxes.HasMessageByHash(h)
}

func (is *ISAAC) ReceiveMessage(m sebakcommon.Message) (ballot Ballot, err error) {
	/*
		Previously the new incoming Message must be checked,
			- TODO `Message` must be saved in `BlockTransactionHistory`
			- TODO check already `IsWellFormed()`
			- TODO check already in BlockTransaction
			- TODO check already in BlockTransactionHistory
	*/

	if is.Boxes.HasMessage(m) {
		err = sebakerror.ErrorNewButKnownMessage
		return
	}

	if ballot, err = NewBallotFromMessage(is.Node.Address(), m); err != nil {
		return
	}

	// self-sign; make new `Ballot` from `Message`
	ballot.SetState(sebakcommon.BallotStateINIT)
	ballot.Vote(VotingYES) // The initial ballot from client will have 'VotingYES'
	ballot.UpdateHash()
	ballot.Sign(is.Node.Keypair(), is.networkID)

	if err = ballot.IsWellFormed(is.networkID); err != nil {
		return
	}

	if _, err = is.Boxes.AddBallot(ballot); err != nil {
		return
	}

	return
}

type BallotStateChange string

const (
	BallotStateNone       BallotStateChange = "none"
	BallotStateChanged    BallotStateChange = "changed"
	BallotStateNotChanged BallotStateChange = "nnnnnnnnnnnnnnnn-changed"
)

func (is *ISAAC) ReceiveBallot(ballot Ballot) (vs VotingStateStaging, err error) {
	switch ballot.State() {
	case sebakcommon.BallotStateINIT:
		vs, err = is.receiveBallotStateINIT(ballot)
	case sebakcommon.BallotStateALLCONFIRM:
		err = sebakerror.ErrorBallotHasInvalidState
		return
	default:
		vs, err = is.receiveBallotVotingStates(ballot)
	}

	if err != nil {
		return
	}

	return
}

func (is *ISAAC) receiveBallotStateINIT(ballot Ballot) (vs VotingStateStaging, err error) {
	var isNew bool

	if isNew, err = is.Boxes.AddBallot(ballot); err != nil {
		return
	}

	if isNew {
		var newBallot Ballot
		newBallot, err = NewBallotFromMessage(is.Node.Keypair().Address(), ballot.Data().Message())
		if err != nil {
			return
		}

		// self-sign
		newBallot.SetState(sebakcommon.BallotStateINIT)
		newBallot.Vote(VotingYES) // The BallotStateINIT ballot will have 'VotingYES'
		newBallot.UpdateHash()
		newBallot.Sign(is.Node.Keypair(), is.networkID)

		if err = newBallot.IsWellFormed(is.networkID); err != nil {
			return
		}

		if _, err = is.Boxes.AddBallot(newBallot); err != nil {
			return
		}
	}

	vr := is.Boxes.VotingResult(ballot)
	if vr.IsClosed() || !vr.CanGetResult(is.VotingThresholdPolicy) {
		return
	}

	votingHole, state, ended := vr.MakeResult(is.VotingThresholdPolicy)
	if ended {
		if vs, err = vr.ChangeState(votingHole, state); err != nil {
			return
		}

		is.Boxes.WaitingBox.RemoveVotingResult(vr) // TODO detect error
		if !vs.IsClosed() {
			is.Boxes.VotingBox.AddVotingResult(vr) // TODO detect error
		}
	}

	return
}

// AddBallot
//
// NOTE(ISSAC.AddBallot): `ISSAC.AddBallot()` only for self-signed Ballot
func (is *ISAAC) AddBallot(ballot Ballot) (err error) {
	vr := is.Boxes.VotingResult(ballot)
	if vr.IsVoted(ballot) {
		return nil
	}
	_, err = is.Boxes.AddBallot(ballot)
	return
}

func (is *ISAAC) CloseConsensus(ballot Ballot) (err error) {
	log.Debug("consensus of this ballot will be closed", "ballot", ballot.MessageHash())
	if !is.HasMessageByHash(ballot.MessageHash()) {
		return sebakerror.ErrorVotingResultNotInBox
	}

	vr := is.Boxes.VotingResult(ballot)

	is.Boxes.WaitingBox.RemoveVotingResult(vr)  // TODO detect error
	is.Boxes.VotingBox.RemoveVotingResult(vr)   // TODO detect error
	is.Boxes.ReservedBox.RemoveVotingResult(vr) // TODO detect error
	is.Boxes.RemoveVotingResult(vr)             // TODO detect error

	return
}

func (is *ISAAC) receiveBallotVotingStates(ballot Ballot) (vs VotingStateStaging, err error) {
	if _, err = is.Boxes.AddBallot(ballot); err != nil {
		return
	}

	vr := is.Boxes.VotingResult(ballot)
	if vr.IsClosed() || !vr.CanGetResult(is.VotingThresholdPolicy) {
		return
	}

	votingHole, state, ended := vr.MakeResult(is.VotingThresholdPolicy)
	if !ended {
		return
	}

	if vs, err = vr.ChangeState(votingHole, state); err != nil {
		return
	}

	return
}
