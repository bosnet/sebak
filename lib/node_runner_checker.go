package sebak

import (
	"errors"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/error"
	"boscoin.io/sebak/lib/network"
	"boscoin.io/sebak/lib/node"
	logging "github.com/inconshreveable/log15"
)

type NodeRunnerHandleMessageChecker struct {
	sebakcommon.DefaultChecker

	NodeRunner *NodeRunner
	LocalNode  *sebaknode.LocalNode
	NetworkID  []byte
	Message    sebaknetwork.Message

	Transaction Transaction
}

// CheckNodeRunnerHandleMessageTransactionUnmarshal makes `Transaction` from
// incoming `sebaknetwork.Message`.
func CheckNodeRunnerHandleMessageTransactionUnmarshal(c sebakcommon.Checker, args ...interface{}) (err error) {
	checker := c.(*NodeRunnerHandleMessageChecker)

	var tx Transaction
	if tx, err = NewTransactionFromJSON(checker.Message.Data); err != nil {
		return
	}

	if err = tx.IsWellFormed(checker.NetworkID); err != nil {
		return
	}

	checker.Transaction = tx
	checker.NodeRunner.Log().Debug("message is transaction")

	return
}

// CheckNodeRunnerHandleMessageHasTransactionAlready checks transaction is in
// `TransactionPool`.
func CheckNodeRunnerHandleMessageHasTransactionAlready(c sebakcommon.Checker, args ...interface{}) (err error) {
	checker := c.(*NodeRunnerHandleMessageChecker)

	is := checker.NodeRunner.Consensus()
	if is.TransactionPool.Has(checker.Transaction.GetHash()) {
		err = sebakerror.ErrorNewButKnownMessage
		return
	}

	return
}

// CheckNodeRunnerHandleMessageHistory checks transaction is in
// `BlockTransactionHistory`, which has the received transaction recently.
func CheckNodeRunnerHandleMessageHistory(c sebakcommon.Checker, args ...interface{}) (err error) {
	checker := c.(*NodeRunnerHandleMessageChecker)

	var found bool
	if found, err = ExistsBlockTransactionHistory(checker.NodeRunner.Storage(), checker.Transaction.GetHash()); found && err == nil {
		checker.NodeRunner.Log().Debug("found in history", "transction", checker.Transaction.GetHash())
		err = sebakerror.ErrorNewButKnownMessage
		return
	}

	bt := NewTransactionHistoryFromTransaction(checker.Transaction, checker.Message.Data)
	if err = bt.Save(checker.NodeRunner.Storage()); err != nil {
		return
	}

	checker.NodeRunner.Log().Debug("saved in history", "transaction", checker.Transaction.GetHash())

	return
}

// CheckNodeRunnerHandleMessagePushIntoTransactionPool add the incoming
// transactions into `TransactionPool`.
func CheckNodeRunnerHandleMessagePushIntoTransactionPool(c sebakcommon.Checker, args ...interface{}) (err error) {
	checker := c.(*NodeRunnerHandleMessageChecker)

	tx := checker.Transaction
	is := checker.NodeRunner.Consensus()
	is.TransactionPool.Add(tx)

	checker.NodeRunner.Log().Debug("push transaction into transactionPool", "transaction", checker.Transaction.GetHash())

	return
}

// CheckNodeRunnerHandleMessageTransactionBroadcast broadcasts the incoming
// transaction to the other nodes.
func CheckNodeRunnerHandleMessageTransactionBroadcast(c sebakcommon.Checker, args ...interface{}) (err error) {
	checker := c.(*NodeRunnerHandleMessageChecker)

	checker.NodeRunner.Log().Debug("transaction from client will be broadcasted", "transaction", checker.Transaction.GetHash())

	// TODO sender should be excluded
	checker.NodeRunner.ConnectionManager().Broadcast(checker.Transaction)

	return
}

type NodeRunnerHandleBallotChecker struct {
	sebakcommon.DefaultChecker

	NodeRunner         *NodeRunner
	LocalNode          *sebaknode.LocalNode
	NetworkID          []byte
	Message            sebaknetwork.Message
	IsNew              bool
	Ballot             Ballot
	VotingHole         VotingHole
	RoundVote          *RoundVote
	Result             RoundVoteResult
	VotingFinished     bool
	FinishedVotingHole VotingHole

	Log logging.Logger
}

// CheckNodeRunnerHandleBallotUnmarshal makes `Ballot` from sebaknetwork.Message.
func CheckNodeRunnerHandleBallotUnmarshal(c sebakcommon.Checker, args ...interface{}) (err error) {
	checker := c.(*NodeRunnerHandleBallotChecker)

	var ballot Ballot
	if ballot, err = NewBallotFromJSON(checker.Message.Data); err != nil {
		return
	}

	if err = ballot.IsWellFormed(checker.NetworkID); err != nil {
		return
	}

	checker.Ballot = ballot
	checker.Log = checker.Log.New(logging.Ctx{"ballot": checker.Ballot.GetHash(), "state": checker.Ballot.State()})
	checker.Log.Debug("message is verified")

	return
}

// CheckNodeRunnerHandleBallotNotFromKnownValidators checks the incoming ballot
// is from the known validators.
func CheckNodeRunnerHandleBallotNotFromKnownValidators(c sebakcommon.Checker, args ...interface{}) (err error) {
	checker := c.(*NodeRunnerHandleBallotChecker)
	if checker.LocalNode.HasValidators(checker.Ballot.Source()) {
		return
	}

	checker.Log.Debug(
		"ballot from unknown validator",
		"from", checker.Ballot.Source(),
	)

	err = sebakerror.ErrorBallotFromUnknownValidator
	return
}

// CheckNodeRunnerHandleBallotAlreadyFinished checks the incoming ballot in
// valid round.
func CheckNodeRunnerHandleBallotAlreadyFinished(c sebakcommon.Checker, args ...interface{}) (err error) {
	checker := c.(*NodeRunnerHandleBallotChecker)

	round := checker.Ballot.Round()
	if !checker.NodeRunner.Consensus().IsAvailableRound(round) {
		err = sebakerror.ErrorBallotAlreadyFinished
		checker.Log.Debug("ballot already finished", "round", round)
		return
	}

	return
}

// CheckNodeRunnerHandleBallotAlreadyVoted checks the node of ballot voted.
func CheckNodeRunnerHandleBallotAlreadyVoted(c sebakcommon.Checker, args ...interface{}) (err error) {
	checker := c.(*NodeRunnerHandleBallotChecker)
	rr := checker.NodeRunner.Consensus().RunningRounds

	var found bool
	var runningRound *RunningRound
	if runningRound, found = rr[checker.Ballot.Round().Hash()]; !found {
		return
	}

	if runningRound.IsVoted(checker.Ballot) {
		err = sebakerror.ErrorBallotAlreadyVoted
		return
	}

	return
}

// CheckNodeRunnerHandleBallotVote vote by incoming ballot; if the ballot is new
// and the round of ballot is not yet registered, this will make new
// `RunningRound`.
func CheckNodeRunnerHandleBallotVote(c sebakcommon.Checker, args ...interface{}) (err error) {
	checker := c.(*NodeRunnerHandleBallotChecker)

	roundHash := checker.Ballot.Round().Hash()
	rr := checker.NodeRunner.Consensus().RunningRounds

	var isNew bool
	var found bool
	var runningRound *RunningRound
	if runningRound, found = rr[roundHash]; !found {
		proposer := checker.NodeRunner.CalculateProposer(
			checker.Ballot.Round().BlockHeight,
			checker.Ballot.Round().Number,
		)

		runningRound, err = NewRunningRound(proposer, checker.Ballot)
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

// CheckNodeRunnerHandleBallotIsSameProposer checks the incoming ballot has the
// same proposer with the current `RunningRound`.
func CheckNodeRunnerHandleBallotIsSameProposer(c sebakcommon.Checker, args ...interface{}) (err error) {
	checker := c.(*NodeRunnerHandleBallotChecker)

	if checker.VotingHole != VotingNOTYET {
		return
	}

	if checker.Ballot.IsFromProposer() && checker.Ballot.Source() == checker.NodeRunner.Node().Address() {
		return
	}

	rr := checker.NodeRunner.Consensus().RunningRounds
	var runningRound *RunningRound
	var found bool
	if runningRound, found = rr[checker.Ballot.Round().Hash()]; !found {
		err = errors.New("`RunningRound` not found")
		return
	}

	if runningRound.Proposer != checker.Ballot.Proposer() {
		checker.VotingHole = VotingNO
		checker.Log.Debug(
			"ballot has different proposer",
			"proposer", runningRound.Proposer,
			"proposed-proposer", checker.Ballot.Proposer(),
		)
		return
	}

	return
}

// CheckNodeRunnerHandleBallotCheckResult checks the voting result.
func CheckNodeRunnerHandleBallotCheckResult(c sebakcommon.Checker, args ...interface{}) (err error) {
	checker := c.(*NodeRunnerHandleBallotChecker)

	if !checker.Ballot.State().IsValidForVote() {
		return
	}

	result, votingHole, finished := checker.RoundVote.CanGetVotingResult(
		checker.NodeRunner.Consensus().VotingThresholdPolicy,
		checker.Ballot.State(),
	)

	checker.Result = result
	checker.VotingFinished = finished
	checker.FinishedVotingHole = votingHole

	if checker.VotingFinished {
		checker.Log.Debug("get result", "finished VotingHole", checker.FinishedVotingHole, "result", checker.Result)
	}

	return
}

var handleBallotTransactionCheckerFuncs = []sebakcommon.CheckerFunc{
	CheckNodeRunnerHandleTransactionsIsNew,
	CheckNodeRunnerHandleTransactionsGetMissingTransaction,
	CheckNodeRunnerHandleTransactionsSameSource,
	CheckNodeRunnerHandleTransactionsSourceCheck,
}

// CheckNodeRunnerHandleINITBallotValidateTransactions validates the
// transactions of newly added ballot.
func CheckNodeRunnerHandleINITBallotValidateTransactions(c sebakcommon.Checker, args ...interface{}) (err error) {
	checker := c.(*NodeRunnerHandleBallotChecker)

	if !checker.IsNew || checker.VotingFinished {
		return
	}

	if checker.RoundVote.IsVotedByNode(checker.Ballot.State(), checker.LocalNode.Address()) {
		err = sebakerror.ErrorBallotAlreadyVoted
		return
	}

	if checker.VotingHole != VotingNOTYET {
		return
	}

	if checker.Ballot.TransactionsLength() < 1 {
		checker.VotingHole = VotingYES
		return
	}

	transactionsChecker := &NodeRunnerHandleTransactionChecker{
		DefaultChecker: sebakcommon.DefaultChecker{handleBallotTransactionCheckerFuncs},
		NodeRunner:     checker.NodeRunner,
		LocalNode:      checker.LocalNode,
		NetworkID:      checker.NetworkID,
		Transactions:   checker.Ballot.Transactions(),
		VotingHole:     VotingNOTYET,
	}

	err = sebakcommon.RunChecker(transactionsChecker, sebakcommon.DefaultDeferFunc)
	if err != nil {
		if _, ok := err.(sebakcommon.CheckerErrorStop); !ok {
			checker.VotingHole = VotingNO
			checker.Log.Debug("failed to handle transactions of ballot", "error", err)
			err = nil
			return
		}
		err = nil
	}

	if transactionsChecker.VotingHole == VotingNO {
		checker.VotingHole = VotingNO
	} else {
		checker.VotingHole = VotingYES
	}

	return
}

// CheckNodeRunnerHandleINITBallotBroadcast will broadcast the validated INIT
// ballot.
func CheckNodeRunnerHandleINITBallotBroadcast(c sebakcommon.Checker, args ...interface{}) (err error) {
	checker := c.(*NodeRunnerHandleBallotChecker)
	if !checker.IsNew {
		return
	}

	newBallot := checker.Ballot
	newBallot.SetSource(checker.LocalNode.Address())
	newBallot.SetVote(sebakcommon.BallotStateSIGN, checker.VotingHole)
	newBallot.Sign(checker.LocalNode.Keypair(), checker.NetworkID)

	rr := checker.NodeRunner.Consensus().RunningRounds

	var runningRound *RunningRound
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

// CheckNodeRunnerHandleSIGNBallotBroadcast will broadcast the confirmed SIGN
// ballot.
func CheckNodeRunnerHandleSIGNBallotBroadcast(c sebakcommon.Checker, args ...interface{}) (err error) {
	checker := c.(*NodeRunnerHandleBallotChecker)
	if !checker.VotingFinished {
		return
	}

	newBallot := checker.Ballot
	newBallot.SetSource(checker.LocalNode.Address())
	newBallot.SetVote(sebakcommon.BallotStateACCEPT, checker.FinishedVotingHole)
	newBallot.Sign(checker.LocalNode.Keypair(), checker.NetworkID)

	rr := checker.NodeRunner.Consensus().RunningRounds
	var runningRound *RunningRound
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

// CheckNodeRunnerHandleACCEPTBallotStore will store the confirmed ballot to
// `Block`.
func CheckNodeRunnerHandleACCEPTBallotStore(c sebakcommon.Checker, args ...interface{}) (err error) {
	checker := c.(*NodeRunnerHandleBallotChecker)

	if !checker.VotingFinished {
		return
	}

	willStore := checker.FinishedVotingHole == VotingYES
	if checker.FinishedVotingHole == VotingYES {
		// TODO If consensused ballot is not for next waiting block, the node
		// will go into **catchup** status.

		if checker.Ballot.TransactionsLength() < 1 {
			willStore = false
			checker.Log.Debug("ballot was finished, but not stored because empty transactions")
		} else {
			var block Block
			block, err = FinishBallot(
				checker.NodeRunner.Storage(),
				checker.Ballot,
				checker.NodeRunner.Consensus().TransactionPool,
			)
			if err != nil {
				return
			}

			checker.NodeRunner.Consensus().SetLatestConsensusedBlock(block)
			checker.Log.Debug("ballot was stored", "block", block)
		}

		err = sebakcommon.CheckerErrorStop{"ballot got consensus and will be stored"}
	} else {
		err = sebakcommon.CheckerErrorStop{"ballot got consensus"}
	}

	checker.NodeRunner.Consensus().CloseConsensus(
		checker.Ballot.Proposer(),
		checker.Ballot.Round(),
		checker.FinishedVotingHole,
	)
	checker.NodeRunner.CloseConsensus(checker.Ballot.Round(), willStore)

	return
}
