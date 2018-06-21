package sebak

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"boscoin.io/sebak/lib/common"
)

func makeVotingResult(n int, str string) (results map[string]*VotingResult) {
	_, _, baseBallot := makeNewBallot(sebakcommon.BallotStateINIT, VotingYES)
	vr, _ := NewVotingResult(baseBallot)
	results = map[string]*VotingResult{}


	for i := 0; i < n; i++ {
		_, _, ballot := makeNewBallot(sebakcommon.BallotStateINIT, VotingYES)
		vr.Add(ballot)
	}

	for i := 0; i < n; i++ {
		_, _, ballot := makeNewBallot(sebakcommon.BallotStateSIGN, VotingYES)
		vr.Add(ballot)
	}

	for i := 0; i < n; i++ {
		_, _, ballot := makeNewBallot(sebakcommon.BallotStateACCEPT, VotingYES)
		vr.Add(ballot)
	}
	vr.MessageHash = str
	results[str] = vr

	return
}


func TestNewExpiredBallotChecker(t *testing.T) {
	bb := NewBallotBoxes()

	bb.Results = makeVotingResult(3)
	bb.WaitingBox
	bb.VotingBox
	bb.ReservedBox
}

func TestExpiredBallotChecker_TakeSnapshot(t *testing.T) {

}
