package consensus

import (
	"testing"

	"github.com/spikeekips/sebak/lib"
	"github.com/spikeekips/sebak/lib/error"
	"github.com/stellar/go/keypair"
)

func makeNewBallot(state BallotState, vote VotingHole) (*keypair.Full, sebak.Transaction, Ballot) {
	kpNode, _ := keypair.Random()
	tx := sebak.MakeTransaction(1)
	ballot, _ := NewBallotFromMessage(kpNode.Address(), tx)

	ballot.SetState(state)
	ballot.Vote(vote)
	ballot.UpdateHash()
	ballot.Sign(kpNode)

	return kpNode, tx, ballot
}

func TestNewBallot(t *testing.T) {
	kpNode, _ := keypair.Random()
	tx := sebak.MakeTransaction(1)
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
	if ballot.B.Message.GetHash() != tx.GetHash() {
		t.Error("`Ballot.B.Hash` mismatch")
		return
	}
	if ballot.B.State != InitialState {
		t.Error("`Ballot.B.State` is not `InitialState`")
		return
	}
	if ballot.B.VotingHole != VotingNOTYET {
		t.Error("initial `Ballot.B.VotingHole` must be `VotingNOTYET`")
		return
	}
}

func TestBallotSign(t *testing.T) {
	kpNode, _, ballot := makeNewBallot(BallotStateINIT, VotingYES)
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
	kpNode, _, ballot := makeNewBallot(BallotStateINIT, VotingNOTYET)

	var err error

	/*
			err = ballot.IsWellFormed()
			if err.(sebak_error.Error).Code != sebak_error.ErrorSignatureVerificationFailed.Code {
				t.Errorf("error must be `ErrorSignatureVerificationFailed`: %v", err)
				return
			}

		ballot.Sign(kpNode)
	*/

	err = ballot.IsWellFormed()
	if err.(sebak_error.Error).Code != sebak_error.ErrorBallotNoVoting.Code {
		t.Errorf("error must be %v", sebak_error.ErrorBallotNoVoting)
		return
	}

	ballot.Vote(VotingYES)
	err = ballot.IsWellFormed()
	if err.(sebak_error.Error).Code != sebak_error.ErrorHashDoesNotMatch.Code {
		t.Errorf("error must be %v", sebak_error.ErrorHashDoesNotMatch)
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
