package sebak

import (
	"github.com/owlchain/sebak/lib/common"
	"github.com/owlchain/sebak/lib/network"
)

type NodeRunnerHandleMessageChecker struct {
	sebakcommon.DefaultChecker

	NodeRunner  *NodeRunner
	CurrentNode sebakcommon.Node
	NetworkID   []byte
	Message     sebaknetwork.Message

	Transaction Transaction
	Ballot      Ballot
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

func CheckNodeRunnerHandleMessageTransactionHasSameSource(c sebakcommon.Checker, args ...interface{}) (err error) {
	checker := c.(*NodeRunnerHandleMessageChecker)

	incomingTx := checker.Transaction
	isaac := checker.NodeRunner.Consensus().(*ISAAC)

	for _, hash := range isaac.Boxes.VotingBox.Hashes {
		existingTx := isaac.Boxes.Messages[hash].(Transaction)
		if incomingTx.B.Source != existingTx.B.Source {
			continue
		}
		err = sebakcommon.CheckerErrorStop{"stop consensus, because same source transaction already in progress"}
		return
	}

	return
}

func CheckNodeRunnerHandleMessageHistory(c sebakcommon.Checker, args ...interface{}) (err error) {
	checker := c.(*NodeRunnerHandleMessageChecker)

	bt := NewTransactionHistoryFromTransaction(checker.Transaction, checker.Message.Data)
	if err = bt.Save(checker.NodeRunner.Storage()); err != nil {
		return
	}

	checker.NodeRunner.Log().Debug("saved in history", "transaction", checker.Transaction.GetHash())

	return
}

func CheckNodeRunnerHandleMessageISAACReceiveMessage(c sebakcommon.Checker, args ...interface{}) (err error) {
	checker := c.(*NodeRunnerHandleMessageChecker)

	var ballot Ballot
	if ballot, err = checker.NodeRunner.Consensus().ReceiveMessage(checker.Transaction); err != nil {
		return
	}

	checker.Ballot = ballot

	return
}

func CheckNodeRunnerHandleMessageSignBallot(c sebakcommon.Checker, args ...interface{}) (err error) {
	checker := c.(*NodeRunnerHandleMessageChecker)

	// self-sign
	checker.Ballot.Vote(VotingYES)
	checker.Ballot.UpdateHash()
	checker.Ballot.Sign(checker.CurrentNode.Keypair(), checker.NetworkID)

	return
}

func CheckNodeRunnerHandleMessageBroadcast(c sebakcommon.Checker, args ...interface{}) (err error) {
	checker := c.(*NodeRunnerHandleMessageChecker)

	checker.NodeRunner.Log().Debug("ballot from client will be broadcasted", "ballot", checker.Ballot.MessageHash())
	checker.NodeRunner.ConnectionManager().Broadcast(checker.Ballot)

	return
}

// TODO check the ballot from known validators

type NodeRunnerHandleBallotChecker struct {
	sebakcommon.DefaultChecker

	NodeRunner         *NodeRunner
	CurrentNode        sebakcommon.Node
	NetworkID          []byte
	Message            sebaknetwork.Message
	Ballot             Ballot
	IsNew              bool
	VotingStateStaging VotingStateStaging
	VotingHole         VotingHole
	WillBroadcast      bool
}

func (c *NodeRunnerHandleBallotChecker) GetTransaction() (tx Transaction) {
	if c.Ballot.IsEmpty() {
		return
	}

	tx = c.Ballot.Data().Data.(Transaction)
	return
}

func CheckNodeRunnerHandleBallotIsWellformed(c sebakcommon.Checker, args ...interface{}) (err error) {
	checker := c.(*NodeRunnerHandleBallotChecker)

	var ballot Ballot
	if ballot, err = NewBallotFromJSON(checker.Message.Data); err != nil {
		return
	}

	checker.Ballot = ballot

	return
}

func CheckNodeRunnerHandleBallotNotFromKnownValidators(c sebakcommon.Checker, args ...interface{}) (err error) {
	checker := c.(*NodeRunnerHandleBallotChecker)
	if checker.CurrentNode.HasValidators(checker.Ballot.B.NodeKey) {
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

func CheckNodeRunnerHandleBallotCheckIsNew(c sebakcommon.Checker, args ...interface{}) (err error) {
	checker := c.(*NodeRunnerHandleBallotChecker)

	checker.IsNew = !checker.NodeRunner.Consensus().HasMessageByHash(checker.Ballot.MessageHash())

	return
}

func CheckNodeRunnerHandleBallotReceiveBallot(c sebakcommon.Checker, args ...interface{}) (err error) {
	checker := c.(*NodeRunnerHandleBallotChecker)

	var vs VotingStateStaging
	if vs, err = checker.NodeRunner.Consensus().ReceiveBallot(checker.Ballot); err != nil {
		return
	}

	checker.VotingStateStaging = vs

	return
}

func CheckNodeRunnerHandleBallotHistory(c sebakcommon.Checker, args ...interface{}) (err error) {
	checker := c.(*NodeRunnerHandleBallotChecker)

	if !checker.IsNew {
		return
	}

	var raw []byte
	if raw, err = checker.Ballot.Data().Serialize(); err != nil {
		return
	}

	tx := checker.GetTransaction()
	bt := NewTransactionHistoryFromTransaction(tx, raw)
	if err = bt.Save(checker.NodeRunner.Storage()); err != nil {
		return
	}

	checker.NodeRunner.Log().Debug("saved in history from ballot", "transction", tx.GetHash())

	return
}

func CheckNodeRunnerHandleBallotStore(c sebakcommon.Checker, args ...interface{}) (err error) {
	checker := c.(*NodeRunnerHandleBallotChecker)

	if !checker.VotingStateStaging.IsStorable() || !checker.VotingStateStaging.IsClosed() {
		return
	}

	if err = FinishTransaction(checker.NodeRunner.Storage(), checker.Ballot, checker.GetTransaction()); err != nil {
		return
	}

	checker.NodeRunner.Log().Debug(
		"got consensus",
		"ballot", checker.Ballot.MessageHash(),
		"votingResultStaging", checker.VotingStateStaging,
	)

	err = sebakcommon.CheckerErrorStop{"got consensus"}

	return
}

func CheckNodeRunnerHandleBallotIsBroadcastable(c sebakcommon.Checker, args ...interface{}) (err error) {
	checker := c.(*NodeRunnerHandleBallotChecker)

	if checker.VotingStateStaging.IsClosed() {
		err = sebakcommon.CheckerErrorStop{"VotingResult is already closed"}
		return
	}

	if checker.IsNew || checker.VotingStateStaging.IsChanged() {
		checker.WillBroadcast = true
	}

	return
}

func CheckNodeRunnerHandleBallotVotingHole(c sebakcommon.Checker, args ...interface{}) (err error) {
	checker := c.(*NodeRunnerHandleBallotChecker)

	if !checker.VotingStateStaging.IsChanged() {
		return
	}
	if checker.VotingStateStaging.State != sebakcommon.BallotStateSIGN {
		return
	}

	tx := checker.GetTransaction()

	votingHole := VotingYES

	if tx.B.Fee < Amount(BaseFee) {
		checker.NodeRunner.Log().Debug("VotingNO: tx.B.Fee < Amount(BaseFee)")
		votingHole = VotingNO
	}

	// NOTE(CheckNodeRunnerHandleBallotVotingHole): if BlockTransaction was
	// not found by tx.B.Checkpoint, it will be genesis block.
	if votingHole == VotingYES {
		bt, err := GetBlockTransactionByCheckpoint(checker.NodeRunner.Storage(), tx.B.Checkpoint)
		if err == nil {
			// compare the stored BlockTransaction with tx
			if votingHole == VotingYES && tx.B.Source != bt.Source {
				checker.NodeRunner.Log().Debug("VotingNO: tx.B.Source != bt.Source", "tx.B.Source", tx.B.Source, "bt.Source", bt.Source)
				votingHole = VotingNO
			}
		}
	}

	if votingHole == VotingYES {
		if ba, err := GetBlockAccount(checker.NodeRunner.Storage(), tx.B.Source); err != nil {
			checker.NodeRunner.Log().Debug("VotingNO: ", "error", err)
			votingHole = VotingNO
		} else if tx.B.Checkpoint != ba.Checkpoint {
			checker.NodeRunner.Log().Debug(
				"VotingNO: tx.B.Checkpoint != ba.Checkpoint",
				"tx.B.Checkpoint", tx.B.Checkpoint,
				"ba.Checkpoint", ba.Checkpoint,
			)
			votingHole = VotingNO
		} else if tx.TotalAmount(true) > MustAmountFromString(ba.Balance) {
			checker.NodeRunner.Log().Debug(
				"VotingNO: tx.TotalAmount(true) > MustAmountFromString(ba.Balance)",
				"tx.TotalAmount(true)", tx.TotalAmount(true),
				"MustAmountFromString(ba.Balance)", MustAmountFromString(ba.Balance),
			)
			votingHole = VotingNO
		}
	}

	checker.VotingHole = votingHole

	return
}

func CheckNodeRunnerHandleBallotBroadcast(c sebakcommon.Checker, args ...interface{}) (err error) {
	checker := c.(*NodeRunnerHandleBallotChecker)

	if checker.VotingStateStaging.IsClosed() {
		if err := checker.NodeRunner.Consensus().CloseConsensus(checker.Ballot); err != nil {
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
	newBallot.Sign(checker.CurrentNode.Keypair(), checker.NetworkID)

	checker.NodeRunner.Consensus().AddBallot(newBallot)

	checker.NodeRunner.Log().Debug(
		"ballot will be broadcasted",
		"ballot", newBallot.MessageHash(),
		"isNew", checker.IsNew,
	)
	checker.NodeRunner.ConnectionManager().Broadcast(newBallot)

	return
}
