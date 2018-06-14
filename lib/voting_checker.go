package sebak

import (
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/error"
)

type VotingResultChecker struct {
	sebakcommon.DefaultChecker

	VotingResult *VotingResult
	Ballot       Ballot
}

func checkBallotResultValidHash(c sebakcommon.Checker, args ...interface{}) (err error) {
	checker := c.(*VotingResultChecker)
	if checker.Ballot.MessageHash() != checker.VotingResult.MessageHash {
		err = sebakerror.ErrorHashDoesNotMatch
		return
	}

	return
}
