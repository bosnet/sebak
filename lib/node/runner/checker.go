package runner

import (
	"encoding/json"

	"boscoin.io/sebak/lib/consensus/round"

	logging "github.com/inconshreveable/log15"

	"boscoin.io/sebak/lib/ballot"
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
	Ballot             ballot.Ballot
	VotingHole         ballot.VotingHole
	Result             consensus.RoundVoteResult
	VotingFinished     bool
	FinishedVotingHole ballot.VotingHole

	Log logging.Logger
}

// BallotUnmarshal makes `Ballot` from common.NetworkMessage.
func BallotUnmarshal(c common.Checker, args ...interface{}) (err error) {
	checker := c.(*BallotChecker)

	var b ballot.Ballot
	if b, err = ballot.NewBallotFromJSON(checker.Message.Data); err != nil {
		return
	}

	checker.Log = checker.Log.New(logging.Ctx{})

	if err = b.IsWellFormed(checker.NetworkID); err != nil {
		return
	}

	checker.Ballot = b
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

// BallotCheckSYNC performs sync by considering sync condition.
// And to participate in the consensus,
// update the latestblock by referring to the database.
func BallotCheckSYNC(c common.Checker, args ...interface{}) (err error) {
	checker := c.(*BallotChecker)

	is := checker.NodeRunner.Consensus()
	b := checker.Ballot
	if is.LatestConfirmedBlock().Height >= b.Round().BlockHeight { // in consensus, not sync
		return
	}

	if !isBallotAcceptYes(b) {
		return
	}

	if !hasBallotValidProposer(is, b) {
		return
	}

	if is.LatestBallot.H.Hash == "" {
		is.LatestBallot = b
	}

	is.SaveNodeHeight(b.Source(), b.Round().BlockHeight)

	var syncHeight uint64
	var nodeAddrs []string
	syncHeight, nodeAddrs, err = checker.NodeRunner.Consensus().GetSyncInfo()
	if err != nil {
		return err
	}

	defer func() {
		if b.Round().BlockHeight == syncHeight {
			checker.NodeRunner.Log().Debug("is.SetLatestConsensusedBlock(blk)", "syncHeight", syncHeight)
			is.LatestBallot = b
		}
	}()

	if err = updateLatestBlockFromDatabase(is, checker.NodeRunner.storage, checker.NodeRunner.Log()); err != nil {
		return
	}

	latestHeight := is.LatestConfirmedBlock().Height
	if latestHeight < syncHeight-1 { // request sync until syncHeight
		checker.NodeRunner.Log().Debug("latestHeight < syncHeight-1", "latestHeight", latestHeight, "syncHeight", syncHeight)
		is.StartSync(syncHeight, nodeAddrs)
	} else {
		if latestHeight == syncHeight-1 { // finish previous and current height ballot
			var previousBlock block.Block
			previousBlock, err = finishBallot(
				checker.NodeRunner.Storage(),
				is.LatestBallot,
				checker.NodeRunner.Consensus().TransactionPool,
				checker.Log,
				checker.NodeRunner.Log(),
			)
			checker.NodeRunner.Consensus().SetLatestConsensusedBlock(previousBlock)
			if err != nil {
				return
			}
		}

		var block block.Block
		block, err = finishBallot(
			checker.NodeRunner.Storage(),
			checker.Ballot,
			checker.NodeRunner.Consensus().TransactionPool,
			checker.Log,
			checker.NodeRunner.Log(),
		)
		if err != nil {
			return
		}

		checker.NodeRunner.Consensus().SetLatestConsensusedBlock(block)
		checker.LocalNode.SetConsensus()
		checker.NodeRunner.TransitISAACState(b.Round(), ballot.StateALLCONFIRM)
		err = NewCheckerStopCloseConsensus(checker, "ballot got consensus")
		return
	}

	return
}

func isBallotAcceptYes(b ballot.Ballot) bool {
	return b.State() == ballot.StateACCEPT && b.Vote() == ballot.VotingYES
}

func hasBallotValidProposer(is *consensus.ISAAC, b ballot.Ballot) bool {
	return b.Proposer() == is.SelectProposer(b.Round().BlockHeight, b.Round().Number)
}

func updateLatestBlockFromDatabase(is *consensus.ISAAC, st *storage.LevelDBBackend, log logging.Logger) error {
	blk, err := block.GetLatestBlock(st)
	if err != nil {
		return err
	}
	if is.LatestConfirmedBlock().Height < blk.Height {
		log.Info("is.SetLatestConsensusedBlock(blk)", "blk.Height", blk.Height)
		is.SetLatestConsensusedBlock(blk)
	}

	return nil
}

// BallotAlreadyFinished checks the incoming ballot in
// valid round.
func BallotAlreadyFinished(c common.Checker, args ...interface{}) (err error) {
	checker := c.(*BallotChecker)
	ballotRound := checker.Ballot.Round()
	if !checker.NodeRunner.Consensus().IsAvailableRound(ballotRound) {
		err = errors.ErrorBallotAlreadyFinished
		checker.Log.Debug("ballot already finished")
		return
	}

	return
}

// BallotAlreadyVoted checks the node of ballot voted.
func BallotAlreadyVoted(c common.Checker, args ...interface{}) (err error) {
	checker := c.(*BallotChecker)
	if checker.NodeRunner.Consensus().IsVoted(checker.Ballot) {
		err = errors.ErrorBallotAlreadyVoted
	}

	return
}

// BallotVote vote by incoming ballot; if the ballot is new
// and the round of ballot is not yet registered, this will make new
// `RunningRound`.
func BallotVote(c common.Checker, args ...interface{}) (err error) {
	checker := c.(*BallotChecker)

	checker.IsNew, err = checker.NodeRunner.Consensus().Vote(checker.Ballot)
	checker.Log.Debug("ballot voted", "ballot", checker.Ballot, "new", checker.IsNew)

	return
}

// BallotIsSameProposer checks the incoming ballot has the
// same proposer with the current `RunningRound`.
func BallotIsSameProposer(c common.Checker, args ...interface{}) (err error) {
	checker := c.(*BallotChecker)

	if checker.VotingHole != ballot.VotingNOTYET {
		return
	}

	if checker.Ballot.IsFromProposer() && checker.Ballot.Source() == checker.NodeRunner.Node().Address() {
		return
	}

	if !checker.NodeRunner.Consensus().HasRunningRound(checker.Ballot.Round().Index()) {
		err = errors.New("`RunningRound` not found")
		return
	}

	if !checker.NodeRunner.Consensus().HasSameProposer(checker.Ballot) {
		checker.VotingHole = ballot.VotingNO
		checker.Log.Debug("ballot has different proposer", "proposer", checker.Ballot.Proposer())
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

	result, votingHole, finished := checker.NodeRunner.Consensus().CanGetVotingResult(checker.Ballot)

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
	var voted bool
	voted, err = checker.NodeRunner.Consensus().IsVotedByNode(checker.Ballot, checker.LocalNode.Address())
	if voted || err != nil {
		err = errors.ErrorBallotAlreadyVoted
		return
	}

	if checker.VotingHole != ballot.VotingNOTYET {
		return
	}

	if checker.Ballot.TransactionsLength() < 1 {
		checker.VotingHole = ballot.VotingYES
		return
	}

	transactionsChecker := &BallotTransactionChecker{
		DefaultChecker: common.DefaultChecker{Funcs: handleBallotTransactionCheckerFuncs},
		NodeRunner:     checker.NodeRunner,
		LocalNode:      checker.LocalNode,
		NetworkID:      checker.NetworkID,
		Transactions:   checker.Ballot.Transactions(),
		VotingHole:     ballot.VotingNOTYET,
	}

	err = common.RunChecker(transactionsChecker, common.DefaultDeferFunc)
	if err != nil {
		if _, ok := err.(common.CheckerErrorStop); !ok {
			checker.VotingHole = ballot.VotingNO
			checker.Log.Debug("failed to handle transactions of ballot", "error", err)
			err = nil
			return
		}
		err = nil
	}

	if transactionsChecker.VotingHole == ballot.VotingNO {
		checker.VotingHole = ballot.VotingNO
	} else {
		checker.VotingHole = ballot.VotingYES
	}

	return
}

// SIGNBallotBroadcast will broadcast the validated SIGN ballot.
func SIGNBallotBroadcast(c common.Checker, args ...interface{}) (err error) {
	checker := c.(*BallotChecker)
	if !checker.IsNew {
		return
	}

	newBallot := checker.Ballot
	newBallot.SetSource(checker.LocalNode.Address())
	newBallot.SetVote(ballot.StateSIGN, checker.VotingHole)
	newBallot.Sign(checker.LocalNode.Keypair(), checker.NetworkID)

	if !checker.NodeRunner.Consensus().HasRunningRound(checker.Ballot.Round().Index()) {
		err = errors.New("RunningRound not found")
		return

	}
	checker.NodeRunner.ConnectionManager().Broadcast(newBallot)
	checker.Log.Debug("ballot will be broadcasted", "newBallot", newBallot)

	return
}

// TransitStateToSIGN changes ISAACState to SIGN
func TransitStateToSIGN(c common.Checker, args ...interface{}) (err error) {
	checker := c.(*BallotChecker)
	if !checker.IsNew {
		return
	}
	checker.NodeRunner.TransitISAACState(checker.Ballot.Round(), ballot.StateSIGN)

	return
}

// ACCEPTBallotBroadcast will broadcast the confirmed ACCEPT
// ballot.
func ACCEPTBallotBroadcast(c common.Checker, args ...interface{}) (err error) {
	checker := c.(*BallotChecker)
	if !checker.VotingFinished {
		return
	}

	newBallot := checker.Ballot
	newBallot.SetSource(checker.LocalNode.Address())
	newBallot.SetVote(ballot.StateACCEPT, checker.FinishedVotingHole)
	newBallot.Sign(checker.LocalNode.Keypair(), checker.NetworkID)

	if !checker.NodeRunner.Consensus().HasRunningRound(checker.Ballot.Round().Index()) {
		err = errors.New("RunningRound not found")
		return

	}
	checker.NodeRunner.ConnectionManager().Broadcast(newBallot)
	checker.Log.Debug("ballot will be broadcasted", "newBallot", newBallot)

	return
}

// TransitStateToACCEPT changes ISAACState to ACCEPT
func TransitStateToACCEPT(c common.Checker, args ...interface{}) (err error) {
	checker := c.(*BallotChecker)
	if !checker.VotingFinished {
		return
	}
	checker.NodeRunner.TransitISAACState(checker.Ballot.Round(), ballot.StateACCEPT)

	return
}

// FinishedBallotStore will store the confirmed ballot to
// `Block`.
func FinishedBallotStore(c common.Checker, args ...interface{}) (err error) {
	checker := c.(*BallotChecker)

	if !checker.VotingFinished {
		return
	}
	ballotRound := checker.Ballot.Round()
	if checker.FinishedVotingHole == ballot.VotingYES {
		var theBlock block.Block
		theBlock, err = finishBallot(
			checker.NodeRunner.Storage(),
			checker.Ballot,
			checker.NodeRunner.Consensus().TransactionPool,
			checker.Log,
			checker.NodeRunner.Log(),
		)
		if err != nil {
			return
		}

		checker.NodeRunner.Consensus().SetLatestConsensusedBlock(theBlock)
		checker.Log.Debug("ballot was stored", "block", theBlock)
		checker.NodeRunner.TransitISAACState(ballotRound, ballot.StateALLCONFIRM)

		err = NewCheckerStopCloseConsensus(checker, "ballot got consensus and will be stored")
	} else {
		checker.NodeRunner.isaacStateManager.IncreaseRound()
		err = NewCheckerStopCloseConsensus(checker, "ballot got consensus")
	}

	checker.NodeRunner.Consensus().CloseConsensus(
		checker.Ballot.Proposer(),
		ballotRound,
		checker.FinishedVotingHole,
	)

	return
}

func finishBallot(st *storage.LevelDBBackend, b ballot.Ballot, transactionPool *transaction.TransactionPool, log, infoLog logging.Logger) (blk block.Block, err error) {
	var isValid bool
	if isValid, err = isValidRound(st, b.Round(), infoLog); err != nil || !isValid {
		return
	}

	var ts *storage.LevelDBBackend
	if ts, err = st.OpenTransaction(); err != nil {
		return
	}

	blk = block.NewBlockFromBallot(b)
	if err = blk.Save(ts); err != nil {
		return
	}
	log.Debug("NewBlock created", "block", blk)

	infoLog.Info("NewBlock created",
		"height", blk.Height,
		"round", blk.Round.Number,
		"timestamp", blk.Header.Timestamp,
		"total-txs", blk.Round.TotalTxs,
		"proposer", blk.Proposer,
	)

	txHashes := b.B.Proposed.Transactions
	transactions := make([]*transaction.Transaction, 0, len(txHashes))
	for _, hash := range txHashes {
		tx, found := transactionPool.Get(hash)
		if !found {
			err = errors.ErrorTransactionNotFound
			return
		}
		transactions = append(transactions, &tx)
	}

	if err = FinishTransactions(blk, transactions, ts); err != nil {
		ts.Discard()
	}

	if err = ts.Commit(); err != nil {
		ts.Discard()
	}

	return
}

func isValidRound(st *storage.LevelDBBackend, r round.Round, log logging.Logger) (bool, error) {
	var latestBlock block.Block
	latestBlock, err := block.GetLatestBlock(st)
	if err != nil {
		return false, err
	}
	if latestBlock.Height != r.BlockHeight {
		log.Error(
			"ballot height is not equal to latestBlock",
			"in ballot", r.BlockHeight,
			"latest height", latestBlock.Height,
		)
		return false, errors.New("ballot height is not equal to latestBlock")
	}
	if latestBlock.Hash != r.BlockHash {
		log.Error(
			"latest block hash in ballot is not equal to latestBlock",
			"in ballot", r.BlockHash,
			"latest block", latestBlock.Hash,
		)
		return false, errors.New("latest block hash in ballot is not equal to latestBlock")
	}

	return true, nil
}

func FinishTransactions(blk block.Block, transactions []*transaction.Transaction, ts *storage.LevelDBBackend) (err error) {
	for _, tx := range transactions {
		raw, _ := json.Marshal(tx)

		bt := block.NewBlockTransactionFromTransaction(blk.Hash, blk.Height, blk.Confirmed, *tx, raw)
		if err = bt.Save(ts); err != nil {
			return
		}
		for _, op := range tx.B.Operations {
			if err = finishOperation(ts, *tx, op, log); err != nil {
				return
			}
		}

		var baSource *block.BlockAccount
		if baSource, err = block.GetBlockAccount(ts, tx.B.Source); err != nil {
			err = errors.ErrorBlockAccountDoesNotExists
			return
		}

		if err = baSource.Withdraw(tx.TotalAmount(true)); err != nil {
			return
		}

		if err = baSource.Save(ts); err != nil {
			return
		}

	}
	return
}

// finishOperation do finish the task after consensus by the type of each operation.
func finishOperation(st *storage.LevelDBBackend, tx transaction.Transaction, op transaction.Operation, log logging.Logger) (err error) {
	switch op.H.Type {
	case transaction.OperationCreateAccount:
		pop, ok := op.B.(transaction.OperationBodyCreateAccount)
		if !ok {
			return errors.ErrorUnknownOperationType
		}
		return finishOperationCreateAccount(st, tx, pop, log)
	case transaction.OperationPayment:
		pop, ok := op.B.(transaction.OperationBodyPayment)
		if !ok {
			return errors.ErrorUnknownOperationType
		}
		return finishOperationPayment(st, tx, pop, log)
	case transaction.OperationCongressVoting, transaction.OperationCongressVotingResult:
		//Nothing to do
		return
	default:
		err = errors.ErrorUnknownOperationType
		return
	}
}

func finishOperationCreateAccount(st *storage.LevelDBBackend, tx transaction.Transaction, op transaction.OperationBodyCreateAccount, log logging.Logger) (err error) {

	var baSource, baTarget *block.BlockAccount
	if baSource, err = block.GetBlockAccount(st, tx.B.Source); err != nil {
		err = errors.ErrorBlockAccountDoesNotExists
		return
	}
	if baTarget, err = block.GetBlockAccount(st, op.TargetAddress()); err == nil {
		err = errors.ErrorBlockAccountAlreadyExists
		return
	} else {
		err = nil
	}

	baTarget = block.NewBlockAccountLinked(
		op.TargetAddress(),
		op.GetAmount(),
		op.Linked,
	)
	if err = baTarget.Save(st); err != nil {
		return
	}

	log.Debug("new account created", "source", baSource, "target", baTarget)

	return
}

func finishOperationPayment(st *storage.LevelDBBackend, tx transaction.Transaction, op transaction.OperationBodyPayment, log logging.Logger) (err error) {

	var baSource, baTarget *block.BlockAccount
	if baSource, err = block.GetBlockAccount(st, tx.B.Source); err != nil {
		err = errors.ErrorBlockAccountDoesNotExists
		return
	}
	if baTarget, err = block.GetBlockAccount(st, op.TargetAddress()); err != nil {
		err = errors.ErrorBlockAccountDoesNotExists
		return
	}

	if err = baTarget.Deposit(op.GetAmount()); err != nil {
		return
	}
	if err = baTarget.Save(st); err != nil {
		return
	}

	log.Debug("payment done", "source", baSource, "target", baTarget, "amount", op.GetAmount())

	return
}
