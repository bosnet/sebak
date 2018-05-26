package sebak

import (
	"context"
	"errors"

	"github.com/btcsuite/btcutil/base58"
	"github.com/spikeekips/sebak/lib/error"
	"github.com/spikeekips/sebak/lib/network"
	"github.com/spikeekips/sebak/lib/common"
	"github.com/stellar/go/keypair"
)

func CheckTransactionSource(ctx context.Context, target interface{}, args ...interface{}) (context.Context, error) {
	if _, err := keypair.Parse(target.(Transaction).B.Source); err != nil {
		return ctx, sebakerror.ErrorBadPublicAddress
	}

	return ctx, nil
}

func CheckTransactionBaseFee(ctx context.Context, target interface{}, args ...interface{}) (context.Context, error) {
	if int64(target.(Transaction).B.Fee) < BaseFee {
		return ctx, sebakerror.ErrorInvalidFee
	}

	return ctx, nil
}

func CheckTransactionOperationIsWellFormed(ctx context.Context, target interface{}, args ...interface{}) (context.Context, error) {
	tx := target.(Transaction)
	for _, op := range tx.B.Operations {
		if ta := op.B.TargetAddress(); tx.B.Source == ta {
			return ctx, sebakerror.ErrorInvalidOperation
		}
		if err := op.IsWellFormed(); err != nil {
			return ctx, err
		}
	}

	return ctx, nil
}

func CheckTransactionVerifySignature(ctx context.Context, target interface{}, args ...interface{}) (context.Context, error) {
	tx := target.(Transaction)

	kp, err := keypair.Parse(tx.B.Source)
	if err != nil {
		return ctx, err
	}
	err = kp.Verify([]byte(tx.H.Hash), base58.Decode(tx.H.Signature))
	if err != nil {
		return ctx, err
	}
	return ctx, nil
}

func CheckTransactionHashMatch(ctx context.Context, target interface{}, args ...interface{}) (context.Context, error) {
	tx := target.(Transaction)
	if tx.H.Hash != tx.B.MakeHashString() {
		return ctx, sebakerror.ErrorHashDoesNotMatch
	}

	return ctx, nil
}

func CheckNodeRunnerHandleMessageTransactionUnmarshal(ctx context.Context, target interface{}, args ...interface{}) (context.Context, error) {
	message, ok := args[0].(sebaknetwork.Message)
	if !ok {
		return ctx, errors.New("found invalid transaction message")
	}

	var tx Transaction
	var err error
	if tx, err = NewTransactionFromJSON(message.Data); err != nil {
		return ctx, err
	}

	if err = tx.IsWellFormed(); err != nil {
		return ctx, err
	}

	// TODO if failed, save in `BlockTransactionHistory`????
	nr := target.(*NodeRunner)
	nr.Log().Debug("message is transaction")
	return context.WithValue(ctx, "transaction", tx), nil
}

func CheckNodeRunnerHandleMessageISAACReceiveMessage(ctx context.Context, target interface{}, args ...interface{}) (context.Context, error) {
	nr := target.(*NodeRunner)
	tx := ctx.Value("transaction").(Transaction)

	var err error
	var ballot Ballot
	if ballot, err = nr.consensus.ReceiveMessage(tx); err != nil {
		return ctx, err
	}

	return context.WithValue(ctx, "ballot", ballot), nil
}

func CheckNodeRunnerHandleMessageSignBallot(ctx context.Context, target interface{}, args ...interface{}) (context.Context, error) {
	ballot := ctx.Value("ballot").(Ballot)

	currentNode := ctx.Value("currentNode").(*sebakcommon.Validator)

	// self-sign
	ballot.Vote(VotingYES)
	ballot.UpdateHash()
	ballot.Sign(currentNode.Keypair())

	return context.WithValue(ctx, "ballot", ballot), nil
}

func CheckNodeRunnerHandleMessageBroadcast(ctx context.Context, target interface{}, args ...interface{}) (context.Context, error) {
	nr := target.(*NodeRunner)
	ballot := ctx.Value("ballot").(Ballot)

	nr.Log().Debug("ballot from client will be broadcasted", "ballot", ballot.MessageHash())
	nr.ConnectionManager().Broadcast(ballot)

	return ctx, nil
}

func CheckNodeRunnerHandleBallotIsWellformed(ctx context.Context, target interface{}, args ...interface{}) (context.Context, error) {
	message, ok := args[0].(sebaknetwork.Message)
	if !ok {
		return ctx, errors.New("found invalid transaction message")
	}

	var err error
	var ballot Ballot
	if ballot, err = NewBallotFromJSON(message.Data); err != nil {
		return ctx, err
	}

	return context.WithValue(ctx, "ballot", ballot), nil
}

func CheckNodeRunnerHandleBallotCheckIsNew(ctx context.Context, target interface{}, args ...interface{}) (context.Context, error) {
	nr := target.(*NodeRunner)
	ballot, ok := ctx.Value("ballot").(Ballot)
	if !ok {
		return ctx, errors.New("found invalid ballot")
	}

	isNew := !nr.consensus.HasMessageByString(ballot.MessageHash())

	return context.WithValue(ctx, "isNew", isNew), nil
}

func CheckNodeRunnerHandleBallotReceiveBallot(ctx context.Context, target interface{}, args ...interface{}) (context.Context, error) {
	nr := target.(*NodeRunner)
	ballot, ok := ctx.Value("ballot").(Ballot)
	if !ok {
		return ctx, errors.New("found invalid ballot")
	}

	var err error
	var vs VotingStateStaging
	if vs, err = nr.consensus.ReceiveBallot(ballot); err != nil {
		return ctx, err
	}

	return context.WithValue(ctx, "vs", vs), nil
}

func CheckNodeRunnerHandleBallotStore(ctx context.Context, target interface{}, args ...interface{}) (context.Context, error) {
	vs, ok := ctx.Value("vs").(VotingStateStaging)
	if !ok {
		return ctx, nil
	}

	if !vs.IsStorable() {
		return ctx, nil
	}

	// TODO store

	nr := target.(*NodeRunner)
	ballot, _ := ctx.Value("ballot").(Ballot)
	nr.Consensus().CloseConsensus(ballot)

	nr.Log().Debug("got consensus", "ballot", ballot.MessageHash(), "votingResultStaging", vs)

	return ctx, sebakcommon.CheckerErrorStop{"got consensus"}
}

func CheckNodeRunnerHandleBallotBroadcast(ctx context.Context, target interface{}, args ...interface{}) (context.Context, error) {
	var willBroadcast bool

	vs := ctx.Value("vs").(VotingStateStaging)
	isNew := ctx.Value("isNew").(bool)
	if vs.IsClosed() {
		return ctx, sebakcommon.CheckerErrorStop{"VotingResult is already closed"}
	} else if vs.IsChanged() {
		willBroadcast = true
	} else if isNew {
		willBroadcast = true
	}

	if !willBroadcast {
		return ctx, nil
	}

	ballot, _ := ctx.Value("ballot").(Ballot)
	nr := target.(*NodeRunner)

	var newBallot Ballot
	newBallot = ballot.Clone()

	state := ballot.State()
	votingHole := ballot.B.VotingHole
	if vs.IsChanged() {
		state = vs.State
		votingHole = vs.VotingHole
	}

	newBallot.SetState(state)
	newBallot.Vote(votingHole)
	newBallot.Sign(nr.Node().Keypair())

	nr.Consensus().AddBallot(newBallot)

	// TODO state is changed, so broadcast
	nr.Log().Debug("ballot will be broadcasted", "ballot", newBallot.MessageHash(), "isNew", isNew)
	nr.ConnectionManager().Broadcast(newBallot)

	return ctx, nil
}
