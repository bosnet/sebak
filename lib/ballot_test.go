package sebak

import (
	"testing"

	"github.com/stellar/go/keypair"

	"github.com/spikeekips/sebak/lib/common"
	"github.com/spikeekips/sebak/lib/error"
)

func makeNewBallot(state sebakcommon.BallotState, vote VotingHole) (*keypair.Full, Transaction, Ballot) {
	kpNode, tx := MakeTransactions(1)
	ballot, _ := NewBallotFromMessage(kpNode.Address(), tx)

	ballot.SetState(state)
	ballot.Vote(vote)
	ballot.UpdateHash()
	ballot.Sign(kpNode)

	return kpNode, tx, ballot
}

func TestNewBallot(t *testing.T) {
	kpNode, tx := MakeTransactions(1)
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
	ballot.Sign(kpNode)

	if len(ballot.H.Signature) < 1 {
		t.Error("`Ballot.H.Signature` is empty")
		return
	}

	if err := ballot.VerifySignature(); err != nil {
		t.Error(err)
		return
	}
}

func TestBallotVote(t *testing.T) {
	kpNode, _, ballot := makeNewBallot(sebakcommon.BallotStateINIT, VotingNOTYET)

	var err error

	/*
			err = ballot.IsWellFormed()
			if err.(sebakerror.Error).Code != sebakerror.ErrorSignatureVerificationFailed.Code {
				t.Errorf("error must be `ErrorSignatureVerificationFailed`: %v", err)
				return
			}

		ballot.Sign(kpNode)
	*/

	err = ballot.IsWellFormed()
	if err.(*sebakerror.Error).Code != sebakerror.ErrorBallotNoVoting.Code {
		t.Errorf("error must be %v", sebakerror.ErrorBallotNoVoting)
		return
	}

	ballot.Vote(VotingYES)
	err = ballot.IsWellFormed()
	if err.(*sebakerror.Error).Code != sebakerror.ErrorHashDoesNotMatch.Code {
		t.Errorf("error must be %v", sebakerror.ErrorHashDoesNotMatch)
		return
	}

	ballot.UpdateHash()
	ballot.Sign(kpNode)
	err = ballot.IsWellFormed()
	if err != nil {
		t.Errorf("failed to `UpdateHash()`: %v", err)
		return
	}
}

func TestBallotNewBallotFromMessageWithTransaction(t *testing.T) {
	kp, _, ballot := makeNewBallot(sebakcommon.BallotStateINIT, VotingYES)
	ballot.Sign(kp)

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

	if err := newBallot.IsWellFormed(); err != nil {
		t.Error(err)
		return
	}
}
