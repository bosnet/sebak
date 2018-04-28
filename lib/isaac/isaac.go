package consensus

import (
	"github.com/spikeekips/sebak/lib/error"
	"github.com/spikeekips/sebak/lib/storage"
	"github.com/spikeekips/sebak/lib/transport"
	"github.com/spikeekips/sebak/lib/util"
)

type ISAAC struct {
	util.SafeLock

	Node                  Node
	VotingThresholdPolicy VotingThresholdPolicy

	Boxes *BallotBoxes

	Storage   *storage.LevelDBBackend
	Transport transport.Transport
}

func NewISAAC(node Node, votingThresholdPolicy VotingThresholdPolicy, st *storage.LevelDBBackend, tp transport.Transport) (is *ISAAC, err error) {
	is = &ISAAC{
		Node: node,
		VotingThresholdPolicy: votingThresholdPolicy,
		Boxes: NewBallotBoxes(),

		Storage:   st,
		Transport: tp,
	}

	return
}

func (is *ISAAC) ReceiveMessage(m util.Message) (ballot Ballot, err error) {
	/*
		Previously the new incoming Message must be checked,
			- TODO `Message` must be saved in `BlockTransactionHistory`
			- TODO check already `IsWellFormed()`
			- TODO check already in BlockTransaction
			- TODO check already in BlockTransactionHistory
	*/

	if is.Boxes.HasMessage(m) {
		err = sebak_error.ErrorNewButKnownMessage
		return
	}

	if ballot, err = NewBallotFromMessage(is.Node.GetKeypair().Address(), m); err != nil {
		return
	}

	// self-sign; make new `Ballot` from `Message`
	ballot.SetState(BallotStateINIT)
	ballot.Vote(VotingYES) // TODO YES or NO
	ballot.UpdateHash()
	ballot.Sign(is.Node.GetKeypair())

	if err = ballot.IsWellFormed(); err != nil {
		return
	}

	if _, err = is.Boxes.AddBallot(ballot); err != nil {
		return
	}

	return
}

func (is *ISAAC) ReceiveBallot(ballot Ballot) (vr *VotingResult, err error) {
	/*
		TODO Previously the new incoming Ballot must be checked `IsWellFormed()`
	*/

	switch ballot.GetState() {
	case BallotStateINIT:
		vr, err = is.receiveBallotStateINIT(ballot)
	case BallotStateALLCONFIRM:
		err = sebak_error.ErrorBallotHasInvalidState
		return
	default:
		vr, err = is.receiveBallotVotingStates(ballot)
	}

	if err != nil {
		return
	}

	return
}

func (is *ISAAC) receiveBallotStateINIT(ballot Ballot) (vr *VotingResult, err error) {
	var isNew bool

	if isNew, err = is.Boxes.AddBallot(ballot); err != nil {
		return
	}

	if !isNew {
		vr = is.Boxes.GetVotingResult(ballot)
		if vr.IsClosed() || !vr.CanGetResult(is.VotingThresholdPolicy) {
			return
		}

		_, ended := vr.GetResult(is.VotingThresholdPolicy)
		if ended {
			is.Boxes.WaitingBox.RemoveVotingResult(vr) // TODO detect error
			is.Boxes.VotingBox.AddVotingResult(vr)     // TODO detect error
		}

		return
	}

	var newBallot Ballot
	newBallot, err = NewBallotFromMessage(is.Node.GetKeypair().Address(), ballot.GetMessage())
	if err != nil {
		return
	}

	// self-sign
	newBallot.SetState(BallotStateINIT)
	newBallot.Vote(VotingYES) // TODO YES or NO
	newBallot.UpdateHash()
	newBallot.Sign(is.Node.GetKeypair())

	if err = newBallot.IsWellFormed(); err != nil {
		return
	}

	if _, err = is.Boxes.AddBallot(newBallot); err != nil {
		return
	}

	// TODO this ballot should be broadcasted

	vr = is.Boxes.GetVotingResult(ballot)

	return
}

func (is *ISAAC) receiveBallotVotingStates(ballot Ballot) (vr *VotingResult, err error) {
	if _, err = is.Boxes.AddBallot(ballot); err != nil {
		return
	}

	vr = is.Boxes.GetVotingResult(ballot)
	if vr.IsClosed() || !vr.CanGetResult(is.VotingThresholdPolicy) {
		return
	}

	_, ended := vr.GetResult(is.VotingThresholdPolicy)
	if ended {
		return
	}

	// TODO if state reaches `BallotStateALLCONFIRM`, externalize it.

	return
}
