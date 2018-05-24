package sebak

import (
	"context"
	"errors"

	"github.com/btcsuite/btcutil/base58"
	"github.com/spikeekips/sebak/lib/error"
	"github.com/spikeekips/sebak/lib/network"
	"github.com/spikeekips/sebak/lib/util"
	"github.com/stellar/go/keypair"
)

func checkTransactionSource(ctx context.Context, target interface{}, args ...interface{}) (context.Context, error) {
	if _, err := keypair.Parse(target.(Transaction).B.Source); err != nil {
		return ctx, sebakerror.ErrorBadPublicAddress
	}

	return ctx, nil
}

func checkTransactionBaseFee(ctx context.Context, target interface{}, args ...interface{}) (context.Context, error) {
	if int64(target.(Transaction).B.Fee) < BaseFee {
		return ctx, sebakerror.ErrorInvalidFee
	}

	return ctx, nil
}

func checkTransactionOperationIsWellFormed(ctx context.Context, target interface{}, args ...interface{}) (context.Context, error) {
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

func checkTransactionVerifySignature(ctx context.Context, target interface{}, args ...interface{}) (context.Context, error) {
	tx := target.(Transaction)
	err := keypair.MustParse(tx.B.Source).Verify([]byte(tx.H.Hash), base58.Decode(tx.H.Signature))
	if err != nil {
		return ctx, err
	}
	return ctx, nil
}

func checkTransactionHashMatch(ctx context.Context, target interface{}, args ...interface{}) (context.Context, error) {
	tx := target.(Transaction)
	if tx.H.Hash != tx.B.MakeHashString() {
		return ctx, sebakerror.ErrorHashDoesNotMatch
	}

	return ctx, nil
}

func checkNodeRunnerHandleMessageTransactionUnmarshal(ctx context.Context, target interface{}, args ...interface{}) (context.Context, error) {
	message, ok := args[0].(network.Message)
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
	nr.log.Debug("message is transaction")
	return context.WithValue(ctx, "transaction", tx), nil
}

func checkNodeRunnerHandleMessageISAACReceiveMessage(ctx context.Context, target interface{}, args ...interface{}) (context.Context, error) {
	nr := target.(*NodeRunner)
	tx := ctx.Value("transaction").(Transaction)

	var err error
	var ballot Ballot
	if ballot, err = nr.consensusProtocol.ReceiveMessage(tx); err != nil {
		return ctx, err
	}

	return context.WithValue(ctx, "ballot", ballot), nil
}

func checkNodeRunnerHandleMessageSignBallot(ctx context.Context, target interface{}, args ...interface{}) (context.Context, error) {
	ballot := ctx.Value("ballot").(Ballot)

	currentNode := ctx.Value("currentNode").(*util.Validator)

	// self-sign
	ballot.Vote(VotingYES)
	ballot.UpdateHash()
	ballot.Sign(currentNode.Keypair())

	return context.WithValue(ctx, "ballot", ballot), nil
}

func checkNodeRunnerHandleMessageBroadcast(ctx context.Context, target interface{}, args ...interface{}) (context.Context, error) {
	nr := target.(*NodeRunner)
	ballot := ctx.Value("ballot").(Ballot)

	nr.log.Debug("ballot from client will be broadcasted", "ballot", ballot.Message().GetHash())
	nr.connectionManager.Broadcast(ballot)

	return ctx, nil
}

func checkNodeRunnerHandleBallotIsWellformed(ctx context.Context, target interface{}, args ...interface{}) (context.Context, error) {
	message, ok := args[0].(network.Message)
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

func checkNodeRunnerHandleBallotCheckIsNew(ctx context.Context, target interface{}, args ...interface{}) (context.Context, error) {
	nr := target.(*NodeRunner)
	ballot, ok := ctx.Value("ballot").(Ballot)
	if !ok {
		return ctx, errors.New("found invalid ballot")
	}

	isNew := !nr.consensusProtocol.HasMessage(ballot)

	return context.WithValue(ctx, "isNew", isNew), nil
}

func checkNodeRunnerHandleBallotReceiveBallot(ctx context.Context, target interface{}, args ...interface{}) (context.Context, error) {
	nr := target.(*NodeRunner)
	ballot, ok := ctx.Value("ballot").(Ballot)
	if !ok {
		return ctx, errors.New("found invalid ballot")
	}

	var err error
	var vt VotingStateStaging
	if vt, err = nr.consensusProtocol.ReceiveBallot(ballot); err != nil {
		return ctx, err
	}

	return context.WithValue(ctx, "vt", vt), nil
}

func checkNodeRunnerHandleBallotIsClosed(ctx context.Context, target interface{}, args ...interface{}) (context.Context, error) {
	vt := ctx.Value("vt").(VotingStateStaging)
	isNew := ctx.Value("isNew").(bool)

	var willBroadcast, willStore bool
	ctx = context.WithValue(ctx, "willBroadcast", willBroadcast)
	ctx = context.WithValue(ctx, "willStore", willStore)

	if isNew {
		willBroadcast = true
		ctx = context.WithValue(ctx, "willBroadcast", willBroadcast)
		return ctx, nil
	}

	if vt.IsEmpty() {
		return ctx, util.CheckerErrorStop{}
	}

	if !vt.IsClosed() {
		willBroadcast = true
		ctx = context.WithValue(ctx, "willBroadcast", willBroadcast)
		return ctx, nil
	}

	if !vt.IsStorable() {
		return ctx, util.CheckerErrorStop{}
	}
	willStore = true
	ctx = context.WithValue(ctx, "willStore", willStore)

	return ctx, nil
}

func checkNodeRunnerHandleBallotStore(ctx context.Context, target interface{}, args ...interface{}) (context.Context, error) {
	willStore := ctx.Value("willStore").(bool)
	if !willStore {
		return ctx, nil
	}

	nr := target.(*NodeRunner)
	vt := ctx.Value("vt").(VotingStateStaging)
	ballot, _ := ctx.Value("ballot").(Ballot)
	nr.log.Debug("got consensus", "ballot", ballot, "votingResultStaging", vt)

	return ctx, util.CheckerErrorStop{}
}

func checkNodeRunnerHandleBallotBroadcast(ctx context.Context, target interface{}, args ...interface{}) (context.Context, error) {
	willBroadcast := ctx.Value("willBroadcast").(bool)
	if !willBroadcast {
		return ctx, nil
	}

	nr := target.(*NodeRunner)
	vt := ctx.Value("vt").(VotingStateStaging)
	isNew := ctx.Value("isNew").(bool)
	ballot, ok := ctx.Value("ballot").(Ballot)
	if !ok {
		return ctx, errors.New("found invalid ballot")
	}

	var err error
	var newBallot Ballot
	newBallot, err = NewBallotFromMessage(nr.currentNode.Address(), ballot.Message())
	if err != nil {
		return ctx, err
	}

	state := ballot.State()
	votingHole := ballot.B.VotingHole
	if vt.IsChanged() {
		state = vt.State
		votingHole = vt.VotingHole
	}

	newBallot.SetState(state)
	newBallot.Vote(votingHole)
	newBallot.UpdateHash()
	newBallot.Sign(nr.currentNode.Keypair())

	// TODO state is changed, so broadcast
	nr.log.Debug("ballot will be broadcasted", "ballot", ballot, "isNew", isNew)
	nr.connectionManager.Broadcast(newBallot)

	return ctx, nil
}
