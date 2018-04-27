package isaac

import (
	"testing"

	"github.com/stellar/go/keypair"
)

func makeBallotsWithSameMessageHash(n uint32) (kps []*keypair.Full, ballots []Ballot) {
	baseKpNode, _, baseBallot := makeNewBallot(BallotStateINIT, VotingYES)
	kps = append(kps, baseKpNode)
	ballots = append(ballots, baseBallot)

	for i := 0; i < int(n)-1; i++ {
		kpNode, _, ballot := makeNewBallot(BallotStateINIT, VotingYES)
		ballot.B.Message.Hash = baseBallot.GetMessage().GetHash()
		ballot.UpdateHash()
		ballot.Sign(kpNode)

		kps = append(kps, kpNode)
		ballots = append(ballots, ballot)
	}

	return
}

func TestNewVotingResult(t *testing.T) {
	_, _, ballot := makeNewBallot(BallotStateINIT, VotingYES)

	vr, err := NewVotingResult(ballot)
	if err != nil {
		t.Error(err)
		return
	}
	if len(vr.ID) < 1 {
		t.Error("`VotingResult.ID` is missing")
		return
	}
}

func TestAddVotingResult(t *testing.T) {
	_, _, ballot0 := makeNewBallot(BallotStateINIT, VotingYES)
	kpNode1, _, ballot1 := makeNewBallot(BallotStateINIT, VotingYES)

	vr, _ := NewVotingResult(ballot0)
	if err := vr.Add(ballot1); err == nil {
		t.Error("`VotingResult.Add` must occurr the `ErrorHashDoesNotMatch`")
	}

	ballot1.B.Message.Hash = ballot0.GetMessage().GetHash()
	ballot1.UpdateHash()
	ballot1.Sign(kpNode1)
	if err := vr.Add(ballot1); err != nil {
		t.Error("failed to `VotingResult.Add`")
		return
	}
}

func TestVotingResultCheckThreshold(t *testing.T) {
	var numberOfBallots uint32 = 5
	_, ballots := makeBallotsWithSameMessageHash(numberOfBallots)

	vr, _ := NewVotingResult(ballots[0])
	for _, ballot := range ballots[1:] {
		vr.Add(ballot)
	}

	if vr.CheckThreshold(BallotStateNONE, numberOfBallots) {
		t.Error("`BallotStateNONE` must be `false`")
		return
	}
	if !vr.CheckThreshold(BallotStateINIT, numberOfBallots) {
		t.Error("`BallotStateINIT` must be `true`")
		return
	}
	if vr.CheckThreshold(BallotStateINIT, 0) {
		t.Error("`BallotStateINIT` must be `false`")
		return
	}
	if !vr.CheckThreshold(BallotStateINIT, numberOfBallots-1) {
		t.Error("`BallotStateINIT` must be `false`")
		return
	}
}

func TestVotingResultGetResult(t *testing.T) {
	var numberOfBallots uint32 = 5
	_, ballots := makeBallotsWithSameMessageHash(numberOfBallots)

	vr, _ := NewVotingResult(ballots[0])
	for _, ballot := range ballots[1:] {
		vr.Add(ballot)
	}

	{
		policy, _ := NewDefaultVotingThresholdPolicy(100, 30, 30)
		policy.SetValidators(uint64(numberOfBallots))

		state, passed := vr.GetResult(policy)
		if state != BallotStateINIT {
			t.Errorf("state must be `BallotStateINIT`: %v", state)
			return
		}
		if !passed {
			t.Error("must be passed")
			return
		}
	}

	{
		// too high threshold
		policy, _ := NewDefaultVotingThresholdPolicy(100, 50, 50)
		policy.SetValidators(uint64(numberOfBallots) + 100)

		state, passed := vr.GetResult(policy)
		if state != BallotStateNONE {
			t.Errorf("state must be `BallotStateNONE`: %v", state)
			return
		}
		if passed {
			t.Error("must not be passed")
			return
		}
	}
}

func TestVotingResultGetResultHigherStateMustBePicked(t *testing.T) {
	var numberOfBallots uint32 = 5
	kps, ballots := makeBallotsWithSameMessageHash(numberOfBallots)

	vr, _ := NewVotingResult(ballots[0])
	for _, ballot := range ballots[1:] {
		vr.Add(ballot)
	}

	// move to `BallotStateACCEPT`
	for i, ballot := range ballots {
		ballot.B.State = BallotStateACCEPT
		ballot.UpdateHash()
		ballot.Sign(kps[i])

		vr.Add(ballot)
	}

	{
		policy, _ := NewDefaultVotingThresholdPolicy(100, 50, 50)
		policy.SetValidators(uint64(numberOfBallots))

		state, passed := vr.GetResult(policy)
		if state != BallotStateACCEPT {
			t.Error("state must be `BallotStateACCEPT`")
			return
		}
		if !passed {
			t.Error("must be passed")
			return
		}
	}
}
