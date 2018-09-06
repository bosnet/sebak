package sebak

import (
	"errors"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/error"
	"boscoin.io/sebak/lib/network"
	"boscoin.io/sebak/lib/node"
	logging "github.com/inconshreveable/log15"
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

func (c CheckerStopCloseConsensus) Checker() sebakcommon.Checker {
	return c.checker
}

type NodeRunnerHandleMessageChecker struct {
	sebakcommon.DefaultChecker

	NodeRunner *NodeRunner
	LocalNode  *sebaknode.LocalNode
	NetworkID  []byte
	Message    network.Message

	Transaction Transaction
}

// CheckNodeRunnerHandleMessageTransactionUnmarshal makes `Transaction` from
// incoming `network.Message`.
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

	checker.NodeRunner.Log().Debug("transaction will be broadcasted", "transaction", checker.Transaction.GetHash())

	// TODO sender should be excluded
	checker.NodeRunner.ConnectionManager().Broadcast(checker.Transaction)

	return
}

type BallotChecker struct {
	sebakcommon.DefaultChecker

	NodeRunner         *NodeRunner
	LocalNode          *sebaknode.LocalNode
	NetworkID          []byte
	Message            network.Message
	IsNew              bool
	Ballot             Ballot
	VotingHole         sebakcommon.VotingHole
	RoundVote          *RoundVote
	Result             RoundVoteResult
	VotingFinished     bool
	FinishedVotingHole sebakcommon.VotingHole

	Log logging.Logger
}

// BallotUnmarshal makes `Ballot` from network.Message.
func BallotUnmarshal(c sebakcommon.Checker, args ...interface{}) (err error) {
	checker := c.(*BallotChecker)

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

// BallotNotFromKnownValidators checks the incoming ballot
// is from the known validators.
func BallotNotFromKnownValidators(c sebakcommon.Checker, args ...interface{}) (err error) {
	checker := c.(*BallotChecker)
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

// BallotAlreadyFinished checks the incoming ballot in
// valid round.
func BallotAlreadyFinished(c sebakcommon.Checker, args ...interface{}) (err error) {
	checker := c.(*BallotChecker)

	round := checker.Ballot.Round()
	if !checker.NodeRunner.Consensus().IsAvailableRound(round) {
		err = sebakerror.ErrorBallotAlreadyFinished
		checker.Log.Debug("ballot already finished", "round", round)
		return
	}

	return
}

// BallotAlreadyVoted checks the node of ballot voted.
func BallotAlreadyVoted(c sebakcommon.Checker, args ...interface{}) (err error) {
	checker := c.(*BallotChecker)
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

// BallotVote vote by incoming ballot; if the ballot is new
// and the round of ballot is not yet registered, this will make new
// `RunningRound`.
func BallotVote(c sebakcommon.Checker, args ...interface{}) (err error) {
	checker := c.(*BallotChecker)

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

// BallotIsSameProposer checks the incoming ballot has the
// same proposer with the current `RunningRound`.
func BallotIsSameProposer(c sebakcommon.Checker, args ...interface{}) (err error) {
	checker := c.(*BallotChecker)

	if checker.VotingHole != sebakcommon.VotingNOTYET {
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
		checker.VotingHole = sebakcommon.VotingNO
		checker.Log.Debug(
			"ballot has different proposer",
			"proposer", runningRound.Proposer,
			"proposed-proposer", checker.Ballot.Proposer(),
		)
		return
	}

	return
}

// BallotCheckResult checks the voting result.
func BallotCheckResult(c sebakcommon.Checker, args ...interface{}) (err error) {
	checker := c.(*BallotChecker)

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
	IsNew,
	GetMissingTransaction,
	BallotTransactionsSameSource,
	BallotTransactionsSourceCheck,
}

// INITBallotValidateTransactions validates the
// transactions of newly added ballot.
func INITBallotValidateTransactions(c sebakcommon.Checker, args ...interface{}) (err error) {
	checker := c.(*BallotChecker)

	if !checker.IsNew || checker.VotingFinished {
		return
	}

	if checker.RoundVote.IsVotedByNode(checker.Ballot.State(), checker.LocalNode.Address()) {
		err = sebakerror.ErrorBallotAlreadyVoted
		return
	}

	if checker.VotingHole != sebakcommon.VotingNOTYET {
		return
	}

	if checker.Ballot.TransactionsLength() < 1 {
		checker.VotingHole = sebakcommon.VotingYES
		return
	}

	transactionsChecker := &BallotTransactionChecker{
		DefaultChecker: sebakcommon.DefaultChecker{Funcs: handleBallotTransactionCheckerFuncs},
		NodeRunner:     checker.NodeRunner,
		LocalNode:      checker.LocalNode,
		NetworkID:      checker.NetworkID,
		Transactions:   checker.Ballot.Transactions(),
		VotingHole:     sebakcommon.VotingNOTYET,
	}

	err = sebakcommon.RunChecker(transactionsChecker, sebakcommon.DefaultDeferFunc)
	if err != nil {
		if _, ok := err.(sebakcommon.CheckerErrorStop); !ok {
			checker.VotingHole = sebakcommon.VotingNO
			checker.Log.Debug("failed to handle transactions of ballot", "error", err)
			err = nil
			return
		}
		err = nil
	}

	if transactionsChecker.VotingHole == sebakcommon.VotingNO {
		checker.VotingHole = sebakcommon.VotingNO
	} else {
		checker.VotingHole = sebakcommon.VotingYES
	}

	return
}

// INITBallotBroadcast will broadcast the validated INIT
// ballot.
func INITBallotBroadcast(c sebakcommon.Checker, args ...interface{}) (err error) {
	checker := c.(*BallotChecker)
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

// SIGNBallotBroadcast will broadcast the confirmed SIGN
// ballot.
func SIGNBallotBroadcast(c sebakcommon.Checker, args ...interface{}) (err error) {
	checker := c.(*BallotChecker)
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

// ACCEPTBallotStore will store the confirmed ballot to
// `Block`.
func ACCEPTBallotStore(c sebakcommon.Checker, args ...interface{}) (err error) {
	checker := c.(*BallotChecker)

	if !checker.VotingFinished {
		return
	}

	willStore := checker.FinishedVotingHole == sebakcommon.VotingYES
	if checker.FinishedVotingHole == sebakcommon.VotingYES {
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
