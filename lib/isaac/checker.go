package consensus

import (
	"github.com/btcsuite/btcutil/base58"
	"github.com/spikeekips/sebak/lib/error"
)

func checkBallotEmptyNodeKey(target interface{}, args ...interface{}) error {
	ballot := target.(Ballot)
	if len(ballot.B.NodeKey) < 1 {
		return sebak_error.ErrorBallotNoNodeKey
	}
	return nil
}

func checkBallotEmptyHashMatch(target interface{}, args ...interface{}) error {
	ballot := target.(Ballot)
	if base58.Encode(ballot.B.MakeHash()) != ballot.GetHash() {
		return sebak_error.ErrorHashDoesNotMatch
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
		return sebak_error.ErrorBallotNoVoting
	}
	return nil
}

func checkBallotValidState(target interface{}, args ...interface{}) error {
	if target.(Ballot).State() == BallotStateNONE {
		return sebak_error.ErrorInvalidState
	}
	return nil
}

func checkBallotHasMessage(target interface{}, args ...interface{}) error {
	ballot := target.(Ballot)
	if ballot.State() != BallotStateINIT {
		if ballot.B.Message.Message == nil {
			return sebak_error.ErrorBallotHasMessage
		}
		return nil
	}

	if ballot.B.Message.Message == nil {
		return sebak_error.ErrorBallotEmptyMessage
	}
	return nil
}

func checkBallotResultValidHash(target interface{}, args ...interface{}) error {
	votingResult := target.(*VotingResult)
	ballot := args[0].(Ballot)
	if ballot.Message().GetHash() != votingResult.MessageHash {
		return sebak_error.ErrorHashDoesNotMatch
	}
	return nil
}
