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

	vr1.MessageHash = "message1"
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

	vr1.MessageHash = "message2"
	results[vr1.MessageHash] = vr1

	_, _, baseBallotACCEPT := makeNewBallot(sebakcommon.BallotStateACCEPT, VotingYES)
	baseBallotACCEPT.B.NodeKey = "n0"
	vr3, _ := NewVotingResult(baseBallotACCEPT)

	_, _, ballotACCEPT1 := makeNewBallot(sebakcommon.BallotStateACCEPT, VotingYES)
	ballotACCEPT1.B.NodeKey = "n1"
	vr3.Add(ballotACCEPT1)

	_, _, ballotACCEPT2 := makeNewBallot(sebakcommon.BallotStateACCEPT, VotingYES)
	ballotACCEPT2.B.NodeKey = "n2"
	vr3.Add(ballotACCEPT2)

	vr1.MessageHash = "message3"
	results[vr1.MessageHash] = vr1

	return
}

func TestNewBallotExpirer(t *testing.T) {
	bb := NewBallotBoxes()

	bb.Results = makeVotingResult()
}
