package sebak

import (
	"errors"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/error"
	"boscoin.io/sebak/lib/network"
	"boscoin.io/sebak/lib/node"
)

type NodeRunnerHandleMessageChecker struct {
	sebakcommon.DefaultChecker

	NodeRunner *NodeRunner
	LocalNode  sebaknode.Node
	NetworkID  []byte
	Message    sebaknetwork.Message

	Transaction Transaction
}

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

func CheckNodeRunnerHandleMessageHasTransactionAlready(c sebakcommon.Checker, args ...interface{}) (err error) {
	checker := c.(*NodeRunnerHandleMessageChecker)

	is := checker.NodeRunner.Consensus()
	if is.TransactionPool.Has(checker.Transaction.GetHash()) {
		err = sebakerror.ErrorNewButKnownMessage
		return
	}

	return
}

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

func CheckNodeRunnerHandleMessagePushIntoTransactionPool(c sebakcommon.Checker, args ...interface{}) (err error) {
	checker := c.(*NodeRunnerHandleMessageChecker)

	tx := checker.Transaction
	is := checker.NodeRunner.Consensus()
	is.TransactionPool.Add(tx)

	checker.NodeRunner.Log().Debug("push transaction into transactionPool", "transaction", checker.Transaction.GetHash())

	return
}

func CheckNodeRunnerHandleMessageTransactionBroadcast(c sebakcommon.Checker, args ...interface{}) (err error) {
	checker := c.(*NodeRunnerHandleMessageChecker)

	checker.NodeRunner.Log().Debug("transaction from client will be broadcasted", "transaction", checker.Transaction.GetHash())

	// TODO sender should be excluded
	checker.NodeRunner.ConnectionManager().Broadcast(checker.Transaction)

	return
}

type NodeRunnerHandleBallotChecker struct {
	sebakcommon.DefaultChecker

	GenesisBlockCheckpoint string
	NodeRunner             *NodeRunner
	LocalNode              sebaknode.Node
	NetworkID              []byte
	Message                sebaknetwork.Message
	IsNew                  bool
	Ballot                 Ballot
	VotingHole             VotingHole
	WillBroadcast          bool
	RoundVote              RoundVote
}

func CheckNodeRunnerHandleBallotUnmarshal(c sebakcommon.Checker, args ...interface{}) (err error) {
	checker := c.(*NodeRunnerHandleBallotChecker)

	var rb Ballot
	if rb, err = NewBallotFromJSON(checker.Message.Data); err != nil {
		return
	}

	if err = rb.IsWellFormed(checker.NetworkID); err != nil {
		return
	}

	checker.Ballot = rb
	checker.NodeRunner.Log().Debug("message is ballot")

	return
}

func CheckNodeRunnerHandleBallotNotFromKnownValidators(c sebakcommon.Checker, args ...interface{}) (err error) {
	checker := c.(*NodeRunnerHandleBallotChecker)

	localNode := checker.LocalNode.(*sebaknode.LocalNode)
	if localNode.HasValidators(checker.Ballot.Source()) {
		return
	}

	checker.NodeRunner.Log().Debug(
		"ballot from unknown validator",
		"from", checker.Ballot.Source(),
		"ballot", checker.Ballot.GetHash(),
	)

	err = sebakcommon.CheckerErrorStop{"ballot from unknown validator"}
	return
}

func CheckNodeRunnerHandleBallotAlreadyFinished(c sebakcommon.Checker, args ...interface{}) (err error) {
	checker := c.(*NodeRunnerHandleBallotChecker)

	round := checker.Ballot.Round()
	if !checker.NodeRunner.Consensus().IsAvailableRound(round) {
		err = errors.New("ballot: already finished")
		checker.NodeRunner.Log().Debug("ballot already finished", "round", round)
		return
	}

	return
}

func CheckNodeRunnerHandleBallotAlreadyVoted(c sebakcommon.Checker, args ...interface{}) (err error) {
	checker := c.(*NodeRunnerHandleBallotChecker)
	rr := checker.NodeRunner.Consensus().RunningRounds

	var found bool
	var runningRound *RunningRound
	if runningRound, found = rr[checker.Ballot.Round().Hash()]; !found {
		return
	}

	if runningRound.IsVoted(checker.Ballot) {
		err = errors.New("ballot: already voted")
		return
	}
	return
}

func CheckNodeRunnerHandleBallotAddRunningRounds(c sebakcommon.Checker, args ...interface{}) (err error) {
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

		runningRound = NewRunningRound(proposer, checker.Ballot)
		rr[roundHash] = runningRound
		isNew = true
	} else {
		isNew = runningRound.Vote(checker.Ballot)
	}

	checker.IsNew = isNew
	checker.RoundVote, err = runningRound.RoundVote(checker.Ballot.Proposer())
	if err != nil {
		return
	}

	checker.NodeRunner.Log().Debug("ballot voted", "runningRound", runningRound, "new", isNew)

	return
}

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
		checker.NodeRunner.Log().Debug(
			"ballot has different proposer",
			"proposer", runningRound.Proposer,
			"proposed-proposer", checker.Ballot.Proposer(),
		)
		return
	}

	return
}

var handleBallotTransactionCheckerFuncs = []sebakcommon.CheckerFunc{
	CheckNodeRunnerHandleTransactionsIsNew,
	CheckNodeRunnerHandleTransactionsGetMissingTransaction,
	CheckNodeRunnerHandleTransactionsSameSource,
	CheckNodeRunnerHandleTransactionsSourceCheck,
}

func CheckNodeRunnerHandleBallotValidateTransactions(c sebakcommon.Checker, args ...interface{}) (err error) {
	checker := c.(*NodeRunnerHandleBallotChecker)

	if !checker.IsNew {
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
		DefaultChecker:    sebakcommon.DefaultChecker{handleBallotTransactionCheckerFuncs},
		NodeRunner:        checker.NodeRunner,
		LocalNode:         checker.LocalNode,
		NetworkID:         checker.NetworkID,
		Ballot:            checker.Ballot,
		ValidTransactions: []string{},
		VotingHole:        VotingNOTYET,
	}

	err = sebakcommon.RunChecker(transactionsChecker, sebakcommon.DefaultDeferFunc)
	if err != nil {
		if _, ok := err.(sebakcommon.CheckerErrorStop); !ok {
			err = nil
			checker.VotingHole = VotingNO
			checker.NodeRunner.Log().Debug("failed to handle transactions of ballot", "error", err)
			return
		}
		err = nil
	}
	if transactionsChecker.VotingHole != VotingNO && !sebakcommon.IsStringArrayEqual(checker.Ballot.ValidTransactions(), transactionsChecker.ValidTransactions) {
		transactionsChecker.VotingHole = VotingNO
		checker.NodeRunner.Log().Debug(
			"invalid transactions of ballot found",
			"proposed", checker.Ballot.ValidTransactions(),
			"validated", transactionsChecker.ValidTransactions,
		)
	}

	if transactionsChecker.VotingHole == VotingNO {
		checker.VotingHole = VotingNO
	} else {
		checker.VotingHole = VotingYES
	}

	return
}

func CheckNodeRunnerHandleBallotBroadcast(c sebakcommon.Checker, args ...interface{}) (err error) {
	checker := c.(*NodeRunnerHandleBallotChecker)
	if !checker.IsNew {
		return
	}

	newBallot := checker.Ballot
	newBallot.SetSource(checker.LocalNode.Address())
	newBallot.SetVote(checker.VotingHole)
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
	checker.NodeRunner.Log().Debug("ballot will be broadcasted", "ballot", newBallot)

	return
}

func CheckNodeRunnerHandleBallotStore(c sebakcommon.Checker, args ...interface{}) (err error) {
	checker := c.(*NodeRunnerHandleBallotChecker)

	voted, finished := checker.RoundVote.CanGetVotingResult(checker.NodeRunner.Consensus().VotingThresholdPolicy)

	if !finished {
		return
	}

	willStore := voted == VotingYES
	if voted == VotingYES {
		// TODO If consensused ballot is not for next waiting block, the node
		// will go into **catchup** status.

		if checker.Ballot.ValidTransactionsLength() < 1 {
			willStore = false
			checker.NodeRunner.Log().Debug("ballot was finished, but not stored because empty transactions")
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
			checker.NodeRunner.Log().Debug("ballot was stored", "block", block)
		}

		err = sebakcommon.CheckerErrorStop{"ballot got consensus and will be stored"}
	} else {
		err = sebakcommon.CheckerErrorStop{"ballot got consensus"}
	}

	checker.NodeRunner.Consensus().CloseConsensus(
		checker.Ballot.Proposer(),
		checker.Ballot.Round(),
		voted,
	)
	checker.NodeRunner.CloseConsensus(checker.Ballot.Round(), willStore)

	return
}
