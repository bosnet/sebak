package sebak

import (
	"github.com/spikeekips/sebak/lib/error"
	"github.com/spikeekips/sebak/lib/storage"
	"github.com/spikeekips/sebak/lib/util"
)

type ISAAC struct {
	util.SafeLock

	Node                  util.Node
	VotingThresholdPolicy VotingThresholdPolicy

	Boxes *BallotBoxes

	Storage *storage.LevelDBBackend
}

func NewISAAC(node util.Node, votingThresholdPolicy VotingThresholdPolicy) (is *ISAAC, err error) {
	is = &ISAAC{
		Node: node,
		VotingThresholdPolicy: votingThresholdPolicy,
		Boxes: NewBallotBoxes(),
	}

	return
}

func (is *ISAAC) GetNode() util.Node {
	return is.Node
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
		err = sebakerror.ErrorNewButKnownMessage
		return
	}

	if ballot, err = NewBallotFromMessage(is.Node.Address(), m); err != nil {
		return
	}

	// self-sign; make new `Ballot` from `Message`
	ballot.SetState(BallotStateINIT)
	ballot.Vote(VotingYES) // TODO YES or NO
	ballot.UpdateHash()
	ballot.Sign(is.Node.Keypair())

	if err = ballot.IsWellFormed(); err != nil {
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
	/*
		TODO Previously the new incoming Ballot must be checked `IsWellFormed()`
	*/

	switch ballot.State() {
	case BallotStateINIT:
		vs, err = is.receiveBallotStateINIT(ballot)
	case BallotStateALLCONFIRM:
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

	if !isNew {
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
			is.Boxes.VotingBox.AddVotingResult(vr)     // TODO detect error
		}

		return
	}

	var newBallot Ballot
	newBallot, err = NewBallotFromMessage(is.Node.Keypair().Address(), ballot.Message())
	if err != nil {
		return
	}

	// self-sign
	newBallot.SetState(BallotStateINIT)
	newBallot.Vote(VotingYES) // TODO YES or NO
	newBallot.UpdateHash()
	newBallot.Sign(is.Node.Keypair())

	if err = newBallot.IsWellFormed(); err != nil {
		return
	}

	if _, err = is.Boxes.AddBallot(newBallot); err != nil {
		return
	}

	// TODO this ballot should be broadcasted

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
	if ended {
		if vs, err = vr.ChangeState(votingHole, state); err != nil {
			return
		}

		return
	}

	// TODO if state reaches `BallotStateALLCONFIRM`, externalize it.

	return
}
