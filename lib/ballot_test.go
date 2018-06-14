package sebak

import (
	"testing"

	"github.com/stellar/go/keypair"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/error"
)

func makeNewBallot(state sebakcommon.BallotState, vote VotingHole) (*keypair.Full, Transaction, Ballot) {
	kpNode, tx := TestMakeTransaction(networkID, 1)
	ballot, _ := NewBallotFromMessage(kpNode.Address(), tx)

	ballot.SetState(state)
	ballot.Vote(vote)
	ballot.UpdateHash()
	ballot.Sign(kpNode, networkID)

	return kpNode, tx, ballot
}

func TestNewBallot(t *testing.T) {
	kpNode, tx := TestMakeTransaction(networkID, 1)
	ballot, _ := NewBallotFromMessage(kpNode.Address(), tx)

	if len(ballot.H.Hash) < 1 {
		t.Error("`Ballot.H.Hash` is empty")
		return
	}
	if len(ballot.B.NodeKey) < 1 {
		t.Error("`Ballot.B.NodeKey` is empty")
		return
	}
	if len(ballot.H.Signature) > 0 {
		t.Error("`Ballot.H.Signature` is not empty")
		return
	}
	if len(ballot.B.Reason) > 0 {
		t.Error("`Ballot.H.Reason` is not empty")
		return
	}
	if !ballot.Data().Message().Equal(tx) {
		t.Error("`Ballot.B.Hash` mismatch")
		return
	}
	if ballot.B.State != sebakcommon.InitialState {
		t.Error("`Ballot.B.State` is not `InitialState`")
		return
	}
	if ballot.B.VotingHole != VotingNOTYET {
		t.Error("initial `Ballot.B.VotingHole` must be `VotingNOTYET`")
		return
	}
}

func TestBallotSign(t *testing.T) {
	kpNode, _, ballot := makeNewBallot(sebakcommon.BallotStateINIT, VotingYES)
	ballot.Sign(kpNode, networkID)

	if len(ballot.H.Signature) < 1 {
		t.Error("`Ballot.H.Signature` is empty")
		return
	}

	if err := ballot.VerifySignature(networkID); err != nil {
		t.Error(err)
		return
	}
}

func TestBallotVote(t *testing.T) {
	kpNode, _, ballot := makeNewBallot(sebakcommon.BallotStateINIT, VotingNOTYET)

	var err error

	err = ballot.IsWellFormed(networkID)
	if err.(*sebakerror.Error).Code != sebakerror.ErrorBallotNoVoting.Code {
		t.Errorf("error must be %v", sebakerror.ErrorBallotNoVoting)
		return
	}

	ballot.Vote(VotingYES)
	err = ballot.IsWellFormed(networkID)
	if err.(*sebakerror.Error).Code != sebakerror.ErrorHashDoesNotMatch.Code {
		t.Errorf("error must be %v", sebakerror.ErrorHashDoesNotMatch)
		return
	}

	ballot.UpdateHash()
	ballot.Sign(kpNode, networkID)
	err = ballot.IsWellFormed(networkID)
	if err != nil {
		t.Errorf("failed to `UpdateHash()`: %v", err)
		return
	}
}

func TestBallotNewBallotFromMessageWithTransaction(t *testing.T) {
	kp, _, ballot := makeNewBallot(sebakcommon.BallotStateINIT, VotingYES)
	ballot.Sign(kp, networkID)

	jsoned, err := ballot.Serialize()
	if err != nil {
		t.Error(err)
		return
	}

	newBallot, err := NewBallotFromJSON(jsoned)
	if err != nil {
		t.Error(err)
		return
	}

	if err := newBallot.IsWellFormed(networkID); err != nil {
		t.Error(err)
		return
	}
}
