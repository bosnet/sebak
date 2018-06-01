package sebak

import (
	"context"

	"github.com/btcsuite/btcutil/base58"
	"github.com/owlchain/sebak/lib/common"
	"github.com/owlchain/sebak/lib/error"
)

func checkBallotEmptyNodeKey(ctx context.Context, target interface{}, args ...interface{}) (context.Context, error) {
	ballot := target.(Ballot)
	if len(ballot.B.NodeKey) < 1 {
		return ctx, sebakerror.ErrorBallotNoNodeKey
	}
	return ctx, nil
}

func checkBallotEmptyHashMatch(ctx context.Context, target interface{}, args ...interface{}) (context.Context, error) {
	ballot := target.(Ballot)
	if base58.Encode(ballot.B.MakeHash()) != ballot.GetHash() {
		return ctx, sebakerror.ErrorHashDoesNotMatch
	}
	return ctx, nil
}

func checkBallotVerifySignature(ctx context.Context, target interface{}, args ...interface{}) (context.Context, error) {
	networkID := ctx.Value("networkID").([]byte)
	if err := target.(Ballot).VerifySignature(networkID); err != nil {
		return ctx, err
	}
	return ctx, nil
}

func checkBallotNoVoting(ctx context.Context, target interface{}, args ...interface{}) (context.Context, error) {
	if target.(Ballot).B.VotingHole == VotingNOTYET {
		return ctx, sebakerror.ErrorBallotNoVoting
	}
	return ctx, nil
}

func checkBallotValidState(ctx context.Context, target interface{}, args ...interface{}) (context.Context, error) {
	if target.(Ballot).State() == sebakcommon.BallotStateNONE {
		return ctx, sebakerror.ErrorInvalidState
	}
	return ctx, nil
}

func checkBallotHasMessage(ctx context.Context, target interface{}, args ...interface{}) (context.Context, error) {
	ballot := target.(Ballot)
	if ballot.Data().Data == nil {
		return ctx, sebakerror.ErrorBallotEmptyMessage
	}

	return ctx, nil
}

func checkBallotResultValidHash(ctx context.Context, target interface{}, args ...interface{}) (context.Context, error) {
	votingResult := target.(*VotingResult)
	ballot := args[0].(Ballot)
	if ballot.MessageHash() != votingResult.MessageHash {
		return ctx, sebakerror.ErrorHashDoesNotMatch
	}
	return ctx, nil
}
