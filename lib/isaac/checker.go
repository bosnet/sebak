package consensus

import (
	"github.com/btcsuite/btcutil/base58"
	"github.com/spikeekips/sebak/lib/error"
)

func checkBallotEmptyNodeKey(target interface{}, args ...interface{}) error {
	ballot := target.(Ballot)
	if len(ballot.B.NodeKey) < 1 {
		return sebakerror.ErrorBallotNoNodeKey
	}
	return nil
}

func checkBallotEmptyHashMatch(target interface{}, args ...interface{}) error {
	ballot := target.(Ballot)
	if base58.Encode(ballot.B.MakeHash()) != ballot.GetHash() {
		return sebakerror.ErrorHashDoesNotMatch
	}
	return nil
}

func checkBallotVerifySignature(target interface{}, args ...interface{}) error {
	if err := target.(Ballot).VerifySignature(); err != nil {
		return err
	}
	return nil
}

func checkBallotNoVoting(target interface{}, args ...interface{}) error {
	if target.(Ballot).B.VotingHole == VotingNOTYET {
		return sebakerror.ErrorBallotNoVoting
	}
	return nil
}

func checkBallotValidState(target interface{}, args ...interface{}) error {
	if target.(Ballot).State() == BallotStateNONE {
		return sebakerror.ErrorInvalidState
	}
	return nil
}

func checkBallotHasMessage(target interface{}, args ...interface{}) error {
	ballot := target.(Ballot)
	if ballot.State() != BallotStateINIT {
		if ballot.B.Message.Message == nil {
			return sebakerror.ErrorBallotHasMessage
		}
		return nil
	}

	if ballot.B.Message.Message == nil {
		return sebakerror.ErrorBallotEmptyMessage
	}
	return nil
}

func checkBallotResultValidHash(target interface{}, args ...interface{}) error {
	votingResult := target.(*VotingResult)
	ballot := args[0].(Ballot)
	if ballot.Message().GetHash() != votingResult.MessageHash {
		return sebakerror.ErrorHashDoesNotMatch
	}
	return nil
}
