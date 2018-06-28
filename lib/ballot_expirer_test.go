package sebak

import (
	"boscoin.io/sebak/lib/common"
	"testing"
)

func makeVotingResult() (results map[string]*VotingResult) {
	results = map[string]*VotingResult{}

	_, _, baseBallotINIT := makeNewBallot(sebakcommon.BallotStateINIT, VotingYES)
	baseBallotINIT.B.NodeKey = "n0"
	vr1, _ := NewVotingResult(baseBallotINIT)

	_, _, ballotINIT1 := makeNewBallot(sebakcommon.BallotStateINIT, VotingYES)
	ballotINIT1.B.NodeKey = "n1"
	vr1.Add(ballotINIT1)

	_, _, ballotINIT2 := makeNewBallot(sebakcommon.BallotStateINIT, VotingYES)
	ballotINIT2.B.NodeKey = "n2"
	vr1.Add(ballotINIT2)

	m1 := NewDummyMessage("message1")
	vr1.MessageHash = m1.GetHash()
	results[vr1.MessageHash] = vr1

	_, _, baseBallotSIGN := makeNewBallot(sebakcommon.BallotStateSIGN, VotingYES)
	baseBallotSIGN.B.NodeKey = "n0"
	vr2, _ := NewVotingResult(baseBallotSIGN)

	_, _, ballotSIGN1 := makeNewBallot(sebakcommon.BallotStateSIGN, VotingYES)
	ballotSIGN1.B.NodeKey = "n1"
	vr2.Add(ballotSIGN1)

	_, _, ballotSIGN2 := makeNewBallot(sebakcommon.BallotStateSIGN, VotingYES)
	ballotSIGN2.B.NodeKey = "n2"
	vr2.Add(ballotSIGN2)

	m2 := NewDummyMessage("message2")
	vr2.MessageHash = m2.GetHash()
	results[vr2.MessageHash] = vr2

	_, _, baseBallotACCEPT := makeNewBallot(sebakcommon.BallotStateACCEPT, VotingYES)
	baseBallotACCEPT.B.NodeKey = "n0"
	vr3, _ := NewVotingResult(baseBallotACCEPT)

	_, _, ballotACCEPT1 := makeNewBallot(sebakcommon.BallotStateACCEPT, VotingYES)
	ballotACCEPT1.B.NodeKey = "n1"
	vr3.Add(ballotACCEPT1)

	_, _, ballotACCEPT2 := makeNewBallot(sebakcommon.BallotStateACCEPT, VotingYES)
	ballotACCEPT2.B.NodeKey = "n2"
	vr3.Add(ballotACCEPT2)

	m3 := NewDummyMessage("message3")
	vr3.MessageHash = m3.GetHash()
	results[vr3.MessageHash] = vr3

	return
}

func TestMakePrevHashesFromSrcBox(t *testing.T) {
	bb := NewBallotBoxes()

	bb.Results = makeVotingResult()

	m1 := NewDummyMessage("message1")
	bb.WaitingBox.Hashes[m1.GetHash()] = true
	bb.Messages[m1.GetHash()] = m1
	bb.Sources[m1.Source()] = m1.GetHash()

	m2 := NewDummyMessage("message2")
	bb.WaitingBox.Hashes[m2.GetHash()] = true
	bb.Messages[m2.GetHash()] = m2
	bb.Sources[m2.Source()] = m2.GetHash()

	m3 := NewDummyMessage("message3")
	bb.WaitingBox.Hashes[m3.GetHash()] = true
	bb.Messages[m3.GetHash()] = m3
	bb.Sources[m3.Source()] = m3.GetHash()

	bem := NewBallotBoxExpireMover(bb.WaitingBox, bb.ReservedBox, bb.Results, 0)
	bem.makePrevHashesFromSrcBox()
}
