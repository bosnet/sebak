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

func CheckNodeRunnerHandleMessageHistory(c sebakcommon.Checker, args ...interface{}) (err error) {
	checker := c.(*NodeRunnerHandleMessageChecker)

	bt := NewTransactionHistoryFromTransaction(checker.Transaction, checker.Message.Data)
	if err = bt.Save(checker.NodeRunner.Storage()); err != nil {
		return
	}

	checker.NodeRunner.Log().Debug("saved in history", "transction", checker.Transaction.GetHash())

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

type NodeRunnerHandleBallotChecker struct {
	sebakcommon.DefaultChecker

	GenesisBlockCheckpoint string
	NodeRunner             *NodeRunner
	CurrentNode            sebakcommon.Node
	NetworkID              []byte
	Message                sebaknetwork.Message
	Ballot                 Ballot
	IsNew                  bool
	VotingStateStaging     VotingStateStaging
	VotingHole             VotingHole
	WillBroadcast          bool
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
	votingHole := VotingNOTYET
	defer func() {
		checker.VotingHole = votingHole
	}()

	if checker.VotingStateStaging.State != sebakcommon.BallotStateSIGN {
		return
	}

	if !checker.WillBroadcast {
		return
	}

	votingHole = VotingNO

	tx := checker.GetTransaction()
	if tx.B.Fee < Amount(BaseFee) {
		return
	}

	// TODO error like `ErrorRecordDoesNotExist` must be checked, whether record
	// does not exist or not.

	if tx.B.Checkpoint == checker.GenesisBlockCheckpoint { // if tx is first from genesis block
		// if tx is based on genesis account, check the source is genesis
		// account.
		if _, err = GetBlockAccountCheckpoint(
			checker.NodeRunner.Storage(),
			tx.B.Source,
			checker.GenesisBlockCheckpoint,
		); err != nil {
			return
		}
	} else {
		// check, the checkpoint of tx exists
		var bt BlockTransaction
		if bt, err = GetBlockTransactionByCheckpoint(checker.NodeRunner.Storage(), tx.B.Checkpoint); err != nil {
			return
		}
		// check, source is same with the `BlockTransaction` of tx's checkpoint
		if tx.B.Source != bt.Source {
			return
		}
	}

	// check, source exists
	var ba *BlockAccount
	if ba, err = GetBlockAccount(checker.NodeRunner.Storage(), tx.B.Source); err != nil {
		return
	}

	// check, checkpoint is based on latest checkpoint
	if !tx.IsValidCheckpoint(ba.Checkpoint) {
		return
	}

	// get the balance at checkpoint
	var bac BlockAccountCheckpoint
	bac, err = GetBlockAccountCheckpoint(checker.NodeRunner.Storage(), tx.B.Source, tx.B.Checkpoint)
	if err != nil {
		return
	}

	totalAmount := tx.TotalAmount(true)
	// check, have enough balance at checkpoint
	if MustAmountFromString(bac.Balance) < totalAmount {
		return
	}

	// check, have enough balance now
	if totalAmount > MustAmountFromString(ba.Balance) {
		return
	}

	votingHole = VotingYES

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
