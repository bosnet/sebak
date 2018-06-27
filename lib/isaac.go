package sebak

import (
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/error"
	"boscoin.io/sebak/lib/node"
)

type ISAAC struct {
	sebakcommon.SafeLock

	networkID             []byte
	Node                  sebaknode.Node
	VotingThresholdPolicy sebakcommon.VotingThresholdPolicy

	Boxes *BallotBoxes
}

func NewISAAC(networkID []byte, node sebaknode.Node, votingThresholdPolicy sebakcommon.VotingThresholdPolicy) (is *ISAAC, err error) {
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

func (is *ISAAC) GetNode() sebaknode.Node {
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
	ballot.Sign(is.Node.Keypair(), is.networkID)

	if err = ballot.IsWellFormed(is.networkID); err != nil {
		return
	}

	if _, err = is.Boxes.AddBallot(ballot); err != nil {
		return
	}

	return
}

func (is *ISAAC) ReceiveBallot(ballot Ballot) (vs VotingStateStaging, err error) {
	switch ballot.State() {
	case sebakcommon.BallotStateINIT:
		vs, err = is.receiveBallotStateINIT(ballot)
	case sebakcommon.BallotStateALLCONFIRM:
		err = sebakerror.ErrorBallotHasInvalidState
	default:
		vs, err = is.receiveBallotVotingStates(ballot)
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
		newBallot.Sign(is.Node.Keypair(), is.networkID)

		if err = newBallot.IsWellFormed(is.networkID); err != nil {
			return
		}

		if _, err = is.Boxes.AddBallot(newBallot); err != nil {
			return
		}
	}

	vr, err := is.Boxes.VotingResult(ballot)
	if err != nil {
		return
	}

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
			is.Boxes.AddSource(ballot)
		}
	}

	return
}

// AddBallot
//
// NOTE(ISSAC.AddBallot): `ISSAC.AddBallot()` only for self-signed Ballot
func (is *ISAAC) AddBallot(ballot Ballot) (err error) {
	vr, err := is.Boxes.VotingResult(ballot)
	if err != nil {
		return
	}
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

	vr, err := is.Boxes.VotingResult(ballot)
	if err != nil {
		return
	}

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

	if !is.Boxes.VotingBox.HasMessageByHash(ballot.MessageHash()) {
		is.Boxes.AddSource(ballot)
	}

	var vr *VotingResult

	if vr, err = is.Boxes.VotingResult(ballot); err != nil {
		return
	}

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
