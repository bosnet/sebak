package sebak

import (
	"errors"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/error"
	"boscoin.io/sebak/lib/network"
	"boscoin.io/sebak/lib/node"
)

type NodeRunnerRoundHandleMessageChecker struct {
	sebakcommon.DefaultChecker

	NodeRunner *NodeRunnerRound
	LocalNode  sebaknode.Node
	NetworkID  []byte
	Message    sebaknetwork.Message

	Transaction Transaction
	Ballot      Ballot
}

func CheckNodeRunnerRoundHandleMessageTransactionUnmarshal(c sebakcommon.Checker, args ...interface{}) (err error) {
	checker := c.(*NodeRunnerRoundHandleMessageChecker)

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

func CheckNodeRunnerRoundHandleMessageHasTransactionAlready(c sebakcommon.Checker, args ...interface{}) (err error) {
	checker := c.(*NodeRunnerRoundHandleMessageChecker)

	is := checker.NodeRunner.Consensus()
	if _, ok := is.TransactionPool[checker.Transaction.GetHash()]; ok {
		err = sebakerror.ErrorNewButKnownMessage
		return
	}

	return
}

func CheckNodeRunnerRoundHandleMessageHistory(c sebakcommon.Checker, args ...interface{}) (err error) {
	checker := c.(*NodeRunnerRoundHandleMessageChecker)

	if _, err = GetBlockTransactionHistory(checker.NodeRunner.Storage(), checker.Transaction.GetHash()); err == nil {
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

func CheckNodeRunnerRoundHandleMessagePushIntoTransactionPool(c sebakcommon.Checker, args ...interface{}) (err error) {
	checker := c.(*NodeRunnerRoundHandleMessageChecker)

	tx := checker.Transaction
	is := checker.NodeRunner.Consensus()
	is.TransactionPool[tx.GetHash()] = tx
	is.TransactionPoolHashes = append(is.TransactionPoolHashes, tx.GetHash())

	checker.NodeRunner.Log().Debug("push transaction into transactionPool", "transaction", checker.Transaction.GetHash())

	return
}

func CheckNodeRunnerRoundHandleMessageTransactionBroadcast(c sebakcommon.Checker, args ...interface{}) (err error) {
	checker := c.(*NodeRunnerRoundHandleMessageChecker)

	checker.NodeRunner.Log().Debug("transaction from client will be broadcasted", "transaction", checker.Transaction.GetHash())
	checker.NodeRunner.ConnectionManager().Broadcast(checker.Transaction)

	return
}

type NodeRunnerRoundHandleBallotChecker struct {
	sebakcommon.DefaultChecker

	GenesisBlockCheckpoint string
	NodeRunner             *NodeRunnerRound
	LocalNode              sebaknode.Node
	NetworkID              []byte
	Message                sebaknetwork.Message
	Ballot                 Ballot
	IsNew                  bool
	VotingStateStaging     VotingStateStaging
	VotingHole             VotingHole
	WillBroadcast          bool
}

func (c *NodeRunnerRoundHandleBallotChecker) GetTransaction() (tx Transaction) {
	if c.Ballot.IsEmpty() {
		return
	}

	tx = c.Ballot.Data().Data.(Transaction)
	return
}

func CheckNodeRunnerRoundHandleBallotIsWellformed(c sebakcommon.Checker, args ...interface{}) (err error) {
	checker := c.(*NodeRunnerRoundHandleBallotChecker)

	var ballot Ballot
	if ballot, err = NewBallotFromJSON(checker.Message.Data); err != nil {
		return
	}

	checker.Ballot = ballot

	return
}

func CheckNodeRunnerRoundHandleBallotNotFromKnownValidators(c sebakcommon.Checker, args ...interface{}) (err error) {
	checker := c.(*NodeRunnerRoundHandleBallotChecker)
	localNode := checker.LocalNode.(*sebaknode.LocalNode)
	if localNode.HasValidators(checker.Ballot.B.NodeKey) {
		return
	}

	checker.NodeRunner.Log().Debug(
		"ballot from unknown validator",
		"from", checker.Ballot.B.NodeKey,
		"ballot", checker.Ballot.MessageHash(),
	)

	err = sebakcommon.CheckerErrorStop{"ballot from unknown validator"}
	return
}

func CheckNodeRunnerRoundHandleBallotCheckIsNew(c sebakcommon.Checker, args ...interface{}) (err error) {
	checker := c.(*NodeRunnerRoundHandleBallotChecker)

	checker.IsNew = !checker.NodeRunner.Consensus().Boxes.HasMessageByHash(checker.Ballot.MessageHash())

	return
}

func CheckNodeRunnerRoundHandleBallotReceiveBallot(c sebakcommon.Checker, args ...interface{}) (err error) {
	checker := c.(*NodeRunnerRoundHandleBallotChecker)

	var vs VotingStateStaging
	if vs, err = checker.NodeRunner.Consensus().ReceiveBallot(checker.Ballot); err != nil {
		return
	}

	checker.VotingStateStaging = vs

	return
}

func CheckNodeRunnerRoundHandleBallotReachedToSIGN(c sebakcommon.Checker, args ...interface{}) (err error) {
	checker := c.(*NodeRunnerRoundHandleBallotChecker)

	var vr *VotingResult
	if vr, err = checker.NodeRunner.Consensus().Boxes.VotingResult(checker.Ballot); err != nil {
		return
	}
	if vr.State == sebakcommon.BallotStateSIGN {
		err = sebakcommon.CheckerErrorStop{"message is reach to `SIGN`"}
		return
	}

	return
}

func CheckNodeRunnerRoundHandleBallotHistory(c sebakcommon.Checker, args ...interface{}) (err error) {
	checker := c.(*NodeRunnerRoundHandleBallotChecker)

	if !checker.IsNew {
		return
	}

	var raw []byte
	if raw, err = checker.Ballot.Data().Serialize(); err != nil {
		return
	}

	tx := checker.GetTransaction()

	if _, err = GetBlockTransactionHistory(checker.NodeRunner.Storage(), tx.GetHash()); err != nil {
		bt := NewTransactionHistoryFromTransaction(tx, raw)
		if err = bt.Save(checker.NodeRunner.Storage()); err != nil {
			return
		}
		checker.NodeRunner.Log().Debug("saved in history from ballot", "transction", tx.GetHash())
	}

	return
}

func CheckNodeRunnerRoundHandleBallotIsBroadcastable(c sebakcommon.Checker, args ...interface{}) (err error) {
	checker := c.(*NodeRunnerRoundHandleBallotChecker)

	if checker.VotingStateStaging.IsClosed() {
		err = sebakcommon.CheckerErrorStop{"VotingResult is already closed"}
		return
	}

	if checker.IsNew || checker.VotingStateStaging.IsChanged() {
		checker.WillBroadcast = true
	}

	return
}

func CheckNodeRunnerRoundHandleBallotBroadcast(c sebakcommon.Checker, args ...interface{}) (err error) {
	checker := c.(*NodeRunnerRoundHandleBallotChecker)

	if checker.VotingStateStaging.IsClosed() {
		if err := checker.NodeRunner.Consensus().CloseBallotConsensus(checker.Ballot); err != nil {
			checker.NodeRunner.Log().Error("failed to close consensus", "error", err)
		}
		err = sebakcommon.CheckerErrorStop{"VotingResult is already closed"}
		return
	}

	if !checker.WillBroadcast {
		return
	}

	var newBallot Ballot
	newBallot = checker.Ballot.Clone()

	state := checker.Ballot.State()
	votingHole := checker.Ballot.B.VotingHole
	if checker.VotingStateStaging.IsChanged() {
		state = checker.VotingStateStaging.State
		votingHole = checker.VotingStateStaging.VotingHole
	}

	if checker.VotingHole != VotingNOTYET {
		votingHole = checker.VotingHole
	}

	checker.VotingHole = votingHole

	newBallot.SetState(state)
	newBallot.Vote(checker.VotingHole)
	newBallot.Sign(checker.LocalNode.Keypair(), checker.NetworkID)

	checker.NodeRunner.Log().Debug(
		"ballot will be broadcasted",
		"ballot", newBallot.MessageHash(),
		"isNew", checker.IsNew,
	)
	checker.NodeRunner.ConnectionManager().Broadcast(newBallot)

	return
}

type NodeRunnerRoundHandleRoundBallotChecker struct {
	sebakcommon.DefaultChecker

	GenesisBlockCheckpoint string
	NodeRunner             *NodeRunnerRound
	LocalNode              sebaknode.Node
	NetworkID              []byte
	Message                sebaknetwork.Message
	IsNew                  bool
	RoundBallot            RoundBallot
	VotingHole             VotingHole
	WillBroadcast          bool
	RoundVote              RoundVote
}

func CheckNodeRunnerRoundHandleRoundBallotUnmarshal(c sebakcommon.Checker, args ...interface{}) (err error) {
	checker := c.(*NodeRunnerRoundHandleRoundBallotChecker)

	var rb RoundBallot
	if rb, err = NewRoundBallotFromJSON(checker.Message.Data); err != nil {
		return
	}

	if err = rb.IsWellFormed(checker.NetworkID); err != nil {
		return
	}

	checker.RoundBallot = rb
	checker.NodeRunner.Log().Debug("message is round-ballot")

	return
}

func CheckNodeRunnerRoundHandleRoundBallotAlreadyFinished(c sebakcommon.Checker, args ...interface{}) (err error) {
	checker := c.(*NodeRunnerRoundHandleRoundBallotChecker)

	round := checker.RoundBallot.Round()
	if !checker.NodeRunner.Consensus().IsAvailableRound(round) {
		err = errors.New("round-ballot: already finished")
		checker.NodeRunner.Log().Debug("round-ballot already finished", "round", round)
		return
	}

	return
}

func CheckNodeRunnerRoundHandleRoundBallotAlreadyVoted(c sebakcommon.Checker, args ...interface{}) (err error) {
	checker := c.(*NodeRunnerRoundHandleRoundBallotChecker)
	rr := checker.NodeRunner.Consensus().RunningRounds

	var found bool
	var runningRound *RunningRound
	if runningRound, found = rr[checker.RoundBallot.Round().Hash()]; !found {
		return
	}

	if runningRound.IsVoted(checker.RoundBallot) {
		err = errors.New("round-ballot: already voted")
		return
	}
	return
}

func CheckNodeRunnerRoundHandleRoundBallotAddRunningRounds(c sebakcommon.Checker, args ...interface{}) (err error) {
	checker := c.(*NodeRunnerRoundHandleRoundBallotChecker)

	roundHash := checker.RoundBallot.Round().Hash()
	rr := checker.NodeRunner.Consensus().RunningRounds

	var isNew bool
	var found bool
	var runningRound *RunningRound
	if runningRound, found = rr[roundHash]; !found {
		proposer := checker.NodeRunner.CalculateProposer(
			checker.RoundBallot.Round().BlockHeight,
			checker.RoundBallot.Round().Number,
		)

		runningRound = NewRunningRound(proposer, checker.RoundBallot)
		rr[roundHash] = runningRound
		isNew = true
	} else {
		isNew = runningRound.Vote(checker.RoundBallot)
	}

	checker.IsNew = isNew
	checker.RoundVote, err = runningRound.RoundVote(checker.RoundBallot.Proposer())
	if err != nil {
		return
	}

	checker.NodeRunner.Log().Debug("round-ballot voted", "runningRound", runningRound, "new", isNew)

	return
}

func CheckNodeRunnerRoundHandleRoundBallotIsSameProposer(c sebakcommon.Checker, args ...interface{}) (err error) {
	checker := c.(*NodeRunnerRoundHandleRoundBallotChecker)

	if checker.VotingHole != VotingNOTYET {
		return
	}

	if checker.RoundBallot.IsFromProposer() && checker.RoundBallot.Source() == checker.NodeRunner.Node().Address() {
		return
	}

	rr := checker.NodeRunner.Consensus().RunningRounds
	var runningRound *RunningRound
	var found bool
	if runningRound, found = rr[checker.RoundBallot.Round().Hash()]; !found {
		err = errors.New("`RunningRound` not found")
		return
	}

	if runningRound.Proposer != checker.RoundBallot.Proposer() {
		checker.VotingHole = VotingNO
		checker.NodeRunner.Log().Debug(
			"round-ballot has different proposer",
			"proposer", runningRound.Proposer,
			"proposed-proposer", checker.RoundBallot.Proposer(),
		)
		return
	}

	return
}

func CheckNodeRunnerRoundHandleRoundBallotValidateTransactions(c sebakcommon.Checker, args ...interface{}) (err error) {
	checker := c.(*NodeRunnerRoundHandleRoundBallotChecker)

	if !checker.IsNew {
		return
	}

	if checker.VotingHole != VotingNOTYET {
		return
	}

	// TODO check transactions are valid or not
	// TODO check the proposed ValidTransactions is valid

	checker.VotingHole = VotingYES
	return
}

func CheckNodeRunnerRoundHandleRoundBallotBroadcast(c sebakcommon.Checker, args ...interface{}) (err error) {
	checker := c.(*NodeRunnerRoundHandleRoundBallotChecker)
	if !checker.IsNew {
		return
	}

	newRoundBallot := checker.RoundBallot
	newRoundBallot.SetSource(checker.LocalNode.Address())
	newRoundBallot.SetVote(checker.VotingHole)
	newRoundBallot.Sign(checker.LocalNode.Keypair(), checker.NetworkID)

	rr := checker.NodeRunner.Consensus().RunningRounds
	var runningRound *RunningRound
	var found bool
	if runningRound, found = rr[checker.RoundBallot.Round().Hash()]; !found {
		err = errors.New("RunningRound not found")
		return
	}
	runningRound.Vote(newRoundBallot)

	checker.NodeRunner.ConnectionManager().Broadcast(newRoundBallot)
	checker.NodeRunner.Log().Debug("round-ballot will be broadcasted", "round-ballot", newRoundBallot)

	return
}

func CheckNodeRunnerRoundHandleRoundBallotStore(c sebakcommon.Checker, args ...interface{}) (err error) {
	checker := c.(*NodeRunnerRoundHandleRoundBallotChecker)

	voted, finished := checker.RoundVote.CanGetVotingResult(checker.NodeRunner.Consensus().VotingThresholdPolicy)

	if !finished {
		return
	}

	willStore := voted == VotingYES
	if voted == VotingYES {
		// TODO If consensused RoundBallot is not for next waiting block, the node
		// will go into **catchup** status.

		if checker.RoundBallot.ValidTransactionsLength() < 1 {
			willStore = false
			checker.NodeRunner.Log().Debug("round-ballot was finished, but not stored because empty transactions")
		} else {
			var block Block
			block, err = FinishRoundBallot(
				checker.NodeRunner.Storage(),
				checker.RoundBallot,
				checker.NodeRunner.Consensus().TransactionPool,
			)
			if err != nil {
				return
			}

			checker.NodeRunner.Consensus().SetLatestConsensusedBlock(block)
			checker.NodeRunner.Log().Debug("round-ballot was stored", "block", block)
		}

		err = sebakcommon.CheckerErrorStop{"round-ballot got consensus and will be stored"}
	} else {
		err = sebakcommon.CheckerErrorStop{"round-ballot got consensus"}
	}

	checker.NodeRunner.Consensus().CloseRoundBallotConsensus(
		checker.RoundBallot.Proposer(),
		checker.RoundBallot.Round(),
		voted,
	)
	checker.NodeRunner.CloseConsensus(checker.RoundBallot.Round(), willStore)

	return
}
