package sebak

import (
	"github.com/btcsuite/btcutil/base58"
	"github.com/owlchain/sebak/lib/common"
	"github.com/owlchain/sebak/lib/error"
)

type BallotChecker struct {
	sebakcommon.DefaultChecker

	Ballot    Ballot
	NetworkID []byte
}

func checkBallotEmptyNodeKey(c sebakcommon.Checker, args ...interface{}) error {
	checker := c.(*BallotChecker)

	if len(checker.Ballot.B.NodeKey) < 1 {
		return sebakerror.ErrorBallotNoNodeKey
	}
	return nil
}

func checkBallotEmptyHashMatch(c sebakcommon.Checker, args ...interface{}) error {
	checker := c.(*BallotChecker)

	if base58.Encode(checker.Ballot.B.MakeHash()) != checker.Ballot.GetHash() {
		return sebakerror.ErrorHashDoesNotMatch
	}
	return nil
}

func checkBallotVerifySignature(c sebakcommon.Checker, args ...interface{}) error {
	checker := c.(*BallotChecker)

	if err := checker.Ballot.VerifySignature(checker.NetworkID); err != nil {
		return err
	}
	return nil
}

func checkBallotNoVoting(c sebakcommon.Checker, args ...interface{}) error {
	checker := c.(*BallotChecker)

	if checker.Ballot.B.VotingHole == VotingNOTYET {
		return sebakerror.ErrorBallotNoVoting
	}
	return nil
}

func checkBallotValidState(c sebakcommon.Checker, args ...interface{}) error {
	checker := c.(*BallotChecker)

	if checker.Ballot.State() == sebakcommon.BallotStateNONE {
		return sebakerror.ErrorInvalidState
	}
	return nil
}

func checkBallotHasMessage(c sebakcommon.Checker, args ...interface{}) error {
	checker := c.(*BallotChecker)

	if checker.Ballot.Data().Data == nil {
		return sebakerror.ErrorBallotEmptyMessage
	}

	return nil
}
