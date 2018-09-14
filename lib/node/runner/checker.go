package runner

import (
	"encoding/json"

	logging "github.com/inconshreveable/log15"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/consensus"
	"boscoin.io/sebak/lib/error"
	"boscoin.io/sebak/lib/node"
	"boscoin.io/sebak/lib/storage"
	"boscoin.io/sebak/lib/transaction"
)

type CheckerStopCloseConsensus struct {
	checker *BallotChecker
	message string
}

func NewCheckerStopCloseConsensus(checker *BallotChecker, message string) CheckerStopCloseConsensus {
	return CheckerStopCloseConsensus{
		checker: checker,
		message: message,
	}
}

func (c CheckerStopCloseConsensus) Error() string {
	return c.message
}

func (c CheckerStopCloseConsensus) Checker() common.Checker {
	return c.checker
}

type BallotChecker struct {
	common.DefaultChecker

	NodeRunner         *NodeRunner
	LocalNode          *node.LocalNode
	NetworkID          []byte
	Message            common.NetworkMessage
	IsNew              bool
	Ballot             block.Ballot
	VotingHole         common.VotingHole
	RoundVote          *consensus.RoundVote
	Result             consensus.RoundVoteResult
	VotingFinished     bool
	FinishedVotingHole common.VotingHole

	Log logging.Logger
}

// BallotUnmarshal makes `Ballot` from common.NetworkMessage.
func BallotUnmarshal(c common.Checker, args ...interface{}) (err error) {
	checker := c.(*BallotChecker)

	var ballot block.Ballot
	if ballot, err = block.NewBallotFromJSON(checker.Message.Data); err != nil {
		return
	}

	if err = ballot.IsWellFormed(checker.NetworkID); err != nil {
		return
	}

	checker.Ballot = ballot
	checker.Log = checker.Log.New(logging.Ctx{
		"ballot":   checker.Ballot.GetHash(),
		"state":    checker.Ballot.State(),
		"proposer": checker.Ballot.Proposer(),
		"round":    checker.Ballot.Round(),
		"from":     checker.Ballot.Source(),
		"vote":     checker.Ballot.Vote(),
	})
	checker.Log.Debug("message is verified")

	return
}

// BallotNotFromKnownValidators checks the incoming ballot
// is from the known validators.
func BallotNotFromKnownValidators(c common.Checker, args ...interface{}) (err error) {
	checker := c.(*BallotChecker)
	if checker.LocalNode.HasValidators(checker.Ballot.Source()) {
		return
	}

	checker.Log.Debug("ballot from unknown validator")

	err = errors.ErrorBallotFromUnknownValidator
	return
}

// BallotAlreadyFinished checks the incoming ballot in
// valid round.
func BallotAlreadyFinished(c common.Checker, args ...interface{}) (err error) {
	checker := c.(*BallotChecker)

	round := checker.Ballot.Round()
	if !checker.NodeRunner.Consensus().IsAvailableRound(round) {
		err = errors.ErrorBallotAlreadyFinished
		checker.Log.Debug("ballot already finished")
		return
	}

	return
}

// BallotAlreadyVoted checks the node of ballot voted.
func BallotAlreadyVoted(c common.Checker, args ...interface{}) (err error) {
	checker := c.(*BallotChecker)
	rr := checker.NodeRunner.Consensus().RunningRounds

	var found bool
	var runningRound *consensus.RunningRound
	if runningRound, found = rr[checker.Ballot.Round().Hash()]; !found {
		return
	}

	if runningRound.IsVoted(checker.Ballot) {
		err = errors.ErrorBallotAlreadyVoted
		return
	}

	return
}

// BallotVote vote by incoming ballot; if the ballot is new
// and the round of ballot is not yet registered, this will make new
// `RunningRound`.
func BallotVote(c common.Checker, args ...interface{}) (err error) {
	checker := c.(*BallotChecker)

	roundHash := checker.Ballot.Round().Hash()
	rr := checker.NodeRunner.Consensus().RunningRounds

	var isNew bool
	var found bool
	var runningRound *consensus.RunningRound
	if runningRound, found = rr[roundHash]; !found {
		proposer := checker.NodeRunner.CalculateProposer(
			checker.Ballot.Round().BlockHeight,
			checker.Ballot.Round().Number,
		)

		runningRound, err = consensus.NewRunningRound(proposer, checker.Ballot)
		if err != nil {
			return
		}

		rr[roundHash] = runningRound
		isNew = true
	} else {
		if _, found = runningRound.Voted[checker.Ballot.Proposer()]; !found {
			isNew = true
		}

		runningRound.Vote(checker.Ballot)
	}

	checker.IsNew = isNew
	checker.RoundVote, err = runningRound.RoundVote(checker.Ballot.Proposer())
	if err != nil {
		return
	}

	checker.Log.Debug("ballot voted", "runningRound", runningRound, "new", isNew)

	return
}

// BallotIsSameProposer checks the incoming ballot has the
// same proposer with the current `RunningRound`.
func BallotIsSameProposer(c common.Checker, args ...interface{}) (err error) {
	checker := c.(*BallotChecker)

	if checker.VotingHole != common.VotingNOTYET {
		return
	}

	if checker.Ballot.IsFromProposer() && checker.Ballot.Source() == checker.NodeRunner.Node().Address() {
		return
	}

	rr := checker.NodeRunner.Consensus().RunningRounds
	var runningRound *consensus.RunningRound
	var found bool
	if runningRound, found = rr[checker.Ballot.Round().Hash()]; !found {
		err = errors.New("`RunningRound` not found")
		return
	}

	if runningRound.Proposer != checker.Ballot.Proposer() {
		checker.VotingHole = common.VotingNO
		checker.Log.Debug("ballot has different proposer", "proposer", runningRound.Proposer)
		return
	}

	return
}

// BallotCheckResult checks the voting result.
func BallotCheckResult(c common.Checker, args ...interface{}) (err error) {
	checker := c.(*BallotChecker)

	if !checker.Ballot.State().IsValidForVote() {
		return
	}

	result, votingHole, finished := checker.RoundVote.CanGetVotingResult(
		checker.NodeRunner.Consensus().VotingThresholdPolicy,
		checker.Ballot.State(),
		checker.Log,
	)

	checker.Result = result
	checker.VotingFinished = finished
	checker.FinishedVotingHole = votingHole

	if checker.VotingFinished {
		checker.Log.Debug(
			"get result",
			"finished VotingHole", checker.FinishedVotingHole,
			"result", checker.Result,
		)
	}

	return
}

var handleBallotTransactionCheckerFuncs = []common.CheckerFunc{
	IsNew,
	GetMissingTransaction,
	BallotTransactionsSameSource,
	BallotTransactionsSourceCheck,
}

// INITBallotValidateTransactions validates the
// transactions of newly added ballot.
func INITBallotValidateTransactions(c common.Checker, args ...interface{}) (err error) {
	checker := c.(*BallotChecker)

	if !checker.IsNew || checker.VotingFinished {
		return
	}

	if checker.RoundVote.IsVotedByNode(checker.Ballot.State(), checker.LocalNode.Address()) {
		err = errors.ErrorBallotAlreadyVoted
		return
	}

	if checker.VotingHole != common.VotingNOTYET {
		return
	}

	if checker.Ballot.TransactionsLength() < 1 {
		checker.VotingHole = common.VotingYES
		return
	}

	transactionsChecker := &BallotTransactionChecker{
		DefaultChecker: common.DefaultChecker{Funcs: handleBallotTransactionCheckerFuncs},
		NodeRunner:     checker.NodeRunner,
		LocalNode:      checker.LocalNode,
		NetworkID:      checker.NetworkID,
		Transactions:   checker.Ballot.Transactions(),
		VotingHole:     common.VotingNOTYET,
	}

	err = common.RunChecker(transactionsChecker, common.DefaultDeferFunc)
	if err != nil {
		if _, ok := err.(common.CheckerErrorStop); !ok {
			checker.VotingHole = common.VotingNO
			checker.Log.Debug("failed to handle transactions of ballot", "error", err)
			err = nil
			return
		}
		err = nil
	}

	if transactionsChecker.VotingHole == common.VotingNO {
		checker.VotingHole = common.VotingNO
	} else {
		checker.VotingHole = common.VotingYES
	}

	return
}

// INITBallotBroadcast will broadcast the validated INIT
// ballot.
func INITBallotBroadcast(c common.Checker, args ...interface{}) (err error) {
	checker := c.(*BallotChecker)
	if !checker.IsNew {
		return
	}

	newBallot := checker.Ballot
	newBallot.SetSource(checker.LocalNode.Address())
	newBallot.SetVote(common.BallotStateSIGN, checker.VotingHole)
	newBallot.Sign(checker.LocalNode.Keypair(), checker.NetworkID)

	rr := checker.NodeRunner.Consensus().RunningRounds

	var runningRound *consensus.RunningRound
	var found bool
	if runningRound, found = rr[checker.Ballot.Round().Hash()]; !found {
		err = errors.New("RunningRound not found")
		return
	}
	runningRound.Vote(newBallot)

	checker.NodeRunner.ConnectionManager().Broadcast(newBallot)
	checker.Log.Debug("ballot will be broadcasted", "newBallot", newBallot)

	return
}

// SIGNBallotBroadcast will broadcast the confirmed SIGN
// ballot.
func SIGNBallotBroadcast(c common.Checker, args ...interface{}) (err error) {
	checker := c.(*BallotChecker)
	if !checker.VotingFinished {
		return
	}

	newBallot := checker.Ballot
	newBallot.SetSource(checker.LocalNode.Address())
	newBallot.SetVote(common.BallotStateACCEPT, checker.FinishedVotingHole)
	newBallot.Sign(checker.LocalNode.Keypair(), checker.NetworkID)

	rr := checker.NodeRunner.Consensus().RunningRounds
	var runningRound *consensus.RunningRound
	var found bool
	if runningRound, found = rr[checker.Ballot.Round().Hash()]; !found {
		err = errors.New("RunningRound not found")
		return
	}
	runningRound.Vote(newBallot)

	checker.NodeRunner.ConnectionManager().Broadcast(newBallot)
	checker.Log.Debug("ballot will be broadcasted", "newBallot", newBallot)

	return
}

// ACCEPTBallotStore will store the confirmed ballot to
// `Block`.
func ACCEPTBallotStore(c common.Checker, args ...interface{}) (err error) {
	checker := c.(*BallotChecker)

	if !checker.VotingFinished {
		return
	}

	willStore := checker.FinishedVotingHole == common.VotingYES
	if checker.FinishedVotingHole == common.VotingYES {
		var theBlock block.Block
		theBlock, err = finishBallot(
			checker.NodeRunner.Storage(),
			checker.Ballot,
			checker.NodeRunner.Consensus().TransactionPool,
			checker.Log,
		)
		if err != nil {
			return
		}

		checker.NodeRunner.Consensus().SetLatestConsensusedBlock(theBlock)
		checker.Log.Debug("ballot was stored", "block", theBlock)

		err = NewCheckerStopCloseConsensus(checker, "ballot got consensus and will be stored")
	} else {
		err = NewCheckerStopCloseConsensus(checker, "ballot got consensus")
	}

	checker.NodeRunner.Consensus().CloseConsensus(
		checker.Ballot.Proposer(),
		checker.Ballot.Round(),
		checker.FinishedVotingHole,
	)
	checker.NodeRunner.CloseConsensus(checker.Ballot, willStore)

	return
}

func finishBallot(st *storage.LevelDBBackend, ballot block.Ballot, transactionPool *transaction.TransactionPool, log logging.Logger) (blk block.Block, err error) {
	var ts *storage.LevelDBBackend
	if ts, err = st.OpenTransaction(); err != nil {
		return
	}

	transactions := map[string]transaction.Transaction{}
	for _, hash := range ballot.B.Proposed.Transactions {
		tx, found := transactionPool.Get(hash)
		if !found {
			err = errors.ErrorTransactionNotFound
			return
		}
		transactions[hash] = tx
	}

	blk = block.NewBlockFromBallot(ballot, log)

	if err = blk.Save(ts); err != nil {
		return
	}

	for _, hash := range ballot.B.Proposed.Transactions {
		tx := transactions[hash]
		raw, _ := json.Marshal(tx)

		bt := block.NewBlockTransactionFromTransaction(blk.Hash, blk.Height, tx, raw)
		if err = bt.Save(ts); err != nil {
			ts.Discard()
			return
		}
		for _, op := range tx.B.Operations {
			if err = finishOperation(ts, tx, op, log); err != nil {
				ts.Discard()
				return
			}
		}

		var baSource *block.BlockAccount
		if baSource, err = block.GetBlockAccount(ts, tx.B.Source); err != nil {
			err = errors.ErrorBlockAccountDoesNotExists
			ts.Discard()
			return
		}

		if err = baSource.Withdraw(tx.TotalAmount(true), tx.NextSequenceID()); err != nil {
			ts.Discard()
			return
		}

		if err = baSource.Save(ts); err != nil {
			ts.Discard()
			return
		}

	}

	if err = ts.Commit(); err != nil {
		ts.Discard()
	}

	return
}

// finishOperation do finish the task after consensus by the type of each operation.
func finishOperation(st *storage.LevelDBBackend, tx transaction.Transaction, op transaction.Operation, log logging.Logger) (err error) {
	switch op.H.Type {
	case transaction.OperationCreateAccount:
		return finishOperationCreateAccount(st, tx, op, log)
	case transaction.OperationPayment:
		return finishOperationPayment(st, tx, op, log)
	case transaction.OperationIssuance:
		return finishOperationIssuance(st, tx, op, log)
	default:
		err = errors.ErrorUnknownOperationType
		return
	}
}

func finishOperationCreateAccount(st *storage.LevelDBBackend, tx transaction.Transaction, op transaction.Operation, log logging.Logger) (err error) {
	var baSource, baTarget *block.BlockAccount
	if baSource, err = block.GetBlockAccount(st, tx.B.Source); err != nil {
		err = errors.ErrorBlockAccountDoesNotExists
		return
	}
	if baTarget, err = block.GetBlockAccount(st, op.B.TargetAddress()); err == nil {
		err = errors.ErrorBlockAccountAlreadyExists
		return
	} else {
		err = nil
	}

	baTarget = block.NewBlockAccount(
		op.B.TargetAddress(),
		op.B.GetAmount(),
	)
	if err = baTarget.Save(st); err != nil {
		return
	}

	log.Debug("new account created", "source", baSource, "target", baTarget)

	return
}

func finishOperationPayment(st *storage.LevelDBBackend, tx transaction.Transaction, op transaction.Operation, log logging.Logger) (err error) {
	var baSource, baTarget *block.BlockAccount
	if baSource, err = block.GetBlockAccount(st, tx.B.Source); err != nil {
		err = errors.ErrorBlockAccountDoesNotExists
		return
	}
	if baTarget, err = block.GetBlockAccount(st, op.B.TargetAddress()); err != nil {
		err = errors.ErrorBlockAccountDoesNotExists
		return
	}

	if err = baTarget.Deposit(op.B.GetAmount()); err != nil {
		return
	}
	if err = baTarget.Save(st); err != nil {
		return
	}

	log.Debug("payment done", "source", baSource, "target", baTarget, "amount", op.B.GetAmount())

	return
}

func finishOperationIssuance(st *storage.LevelDBBackend, tx transaction.Transaction, op transaction.Operation, log logging.Logger) (err error) {
	var baTarget *block.BlockAccount
	if baTarget, err = block.GetBlockAccount(st, op.B.TargetAddress()); err != nil {
		err = errors.ErrorBlockAccountDoesNotExists
		return
	}

	if err = baTarget.Deposit(op.B.GetAmount()); err != nil {
		return
	}
	if err = baTarget.Save(st); err != nil {
		return
	}

	log.Debug("payment done", "target", baTarget, "amount", op.B.GetAmount())

	return
}
