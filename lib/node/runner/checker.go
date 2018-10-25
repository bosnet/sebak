package runner

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"

	logging "github.com/inconshreveable/log15"

	"boscoin.io/sebak/lib/ballot"
	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/consensus"
	"boscoin.io/sebak/lib/error"
	"boscoin.io/sebak/lib/node"
	"boscoin.io/sebak/lib/storage"
	"boscoin.io/sebak/lib/transaction"
	"boscoin.io/sebak/lib/transaction/operation"
	"boscoin.io/sebak/lib/voting"
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

	if err = b.IsWellFormed(checker.NetworkID, checker.NodeRunner.Conf); err != nil {
		return
	}

	checker.Ballot = b
	checker.Log = checker.Log.New(logging.Ctx{
		"ballot":      checker.Ballot.GetHash(),
		"state":       checker.Ballot.State(),
		"proposer":    checker.Ballot.Proposer(),
		"votingBasis": checker.Ballot.VotingBasis(),
		"from":        checker.Ballot.Source(),
		"vote":        checker.Ballot.Vote(),
	})
	checker.Log.Debug("message is verified")

	return
}

// BallotValidateOperationBodyCollectTxFee validates
// `CollectTxFee`.
func BallotValidateOperationBodyCollectTxFee(c common.Checker, args ...interface{}) (err error) {
	checker := c.(*BallotChecker)

	var opb operation.CollectTxFee
	if opb, err = checker.Ballot.ProposerTransaction().CollectTxFee(); err != nil {
		return
	}

	// check common account
	if opb.Target != checker.NodeRunner.CommonAccountAddress {
		err = errors.ErrorInvalidOperation
		return
	}

	return
}

// BallotValidateOperationBodyInflation validates `Inflation`
func BallotValidateOperationBodyInflation(c common.Checker, args ...interface{}) (err error) {
	checker := c.(*BallotChecker)

	var opb operation.Inflation
	if opb, err = checker.Ballot.ProposerTransaction().Inflation(); err != nil {
		return
	}

	// check common account
	if opb.Target != checker.NodeRunner.CommonAccountAddress {
		err = errors.ErrorInvalidOperation
		return
	}
	if opb.InitialBalance != checker.NodeRunner.InitialBalance {
		err = errors.ErrorInvalidOperation
		return
	}

	if opb.Ratio != common.InflationRatioString {
		err = errors.ErrorInvalidOperation
		return
	}

	var expectedInflation common.Amount
	if checker.NodeRunner.Consensus().LatestBlock().Height <= common.BlockHeightEndOfInflation {
		expectedInflation, err = common.CalculateInflation(checker.NodeRunner.InitialBalance)
		if err != nil {
			return
		}
	}

	if opb.Amount != expectedInflation {
		err = errors.ErrorInvalidOperation
		return
	}

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
func BallotCheckSYNC(c common.Checker, args ...interface{}) error {
	checker := c.(*BallotChecker)
	var err error

	is := checker.NodeRunner.Consensus()
	b := checker.Ballot
	latestHeight := is.LatestBlock().Height
	if latestHeight >= b.VotingBasis().Height { // in consensus, not sync
		return nil
	}

	if !isBallotAcceptYes(b) {
		return nil
	}

	if !hasBallotValidProposer(is, b) {
		return nil
	}

	if is.LatestBallot.H.Hash == "" {
		is.LatestBallot = b
	}

	is.SaveNodeHeight(b.Source(), b.VotingBasis().Height)

	var syncHeight uint64
	var nodeAddrs []string
	syncHeight, nodeAddrs, err = checker.NodeRunner.Consensus().GetSyncInfo()
	if err != nil {
		return err
	}

	defer func() {
		if b.VotingBasis().Height == syncHeight {
			is.LatestBallot = b
		}
	}()

	if latestHeight < syncHeight-1 { // request sync until syncHeight
		checker.NodeRunner.Log().Debug("latestHeight < syncHeight-1", "latestHeight", latestHeight, "syncHeight", syncHeight)
		is.StartSync(syncHeight, nodeAddrs)
		return NewCheckerStopCloseConsensus(checker, "ballot makes node in sync")
	} else {
		if latestHeight == syncHeight-1 { // finish previous and current height ballot
			_, err = finishBallot(
				checker.NodeRunner.Storage(),
				is.LatestBallot,
				checker.NodeRunner.TransactionPool,
				checker.Log,
				checker.NodeRunner.Log(),
			)
			if err != nil {
				return err
			}
		}

		_, err = finishBallot(
			checker.NodeRunner.Storage(),
			checker.Ballot,
			checker.NodeRunner.TransactionPool,
			checker.Log,
			checker.NodeRunner.Log(),
		)
		if err != nil {
			return err
		}

		checker.LocalNode.SetConsensus()
		checker.NodeRunner.TransitISAACState(b.VotingBasis(), ballot.StateALLCONFIRM)
		return NewCheckerStopCloseConsensus(checker, "ballot got consensus")
	}
}

func isBallotAcceptYes(b ballot.Ballot) bool {
	return b.State() == ballot.StateACCEPT && b.Vote() == ballot.VotingYES
}

func hasBallotValidProposer(is *consensus.ISAAC, b ballot.Ballot) bool {
	return b.Proposer() == is.SelectProposer(b.VotingBasis().Height, b.VotingBasis().Round)
}

// BallotAlreadyFinished checks the incoming ballot in
// valid round.
func BallotAlreadyFinished(c common.Checker, args ...interface{}) (err error) {
	checker := c.(*BallotChecker)
	ballotRound := checker.Ballot.VotingBasis()
	if !checker.NodeRunner.Consensus().IsAvailableRound(
		ballotRound,
		block.GetLatestBlock(checker.NodeRunner.Storage()),
	) {
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

	if !checker.NodeRunner.Consensus().HasRunningRound(checker.Ballot.VotingBasis().Index()) {
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

// getMissingTransaction will get the missing tranactions, that is, not in
// `TransactionPool` from proposer.
func getMissingTransaction(checker *BallotChecker) (err error) {
	// get missing transactions
	var unknown []string
	for _, hash := range checker.Ballot.Transactions() {
		if checker.NodeRunner.TransactionPool.Has(hash) {
			continue
		}
		unknown = append(unknown, hash)
	}

	if len(unknown) < 1 {
		return
	}

	client := checker.NodeRunner.ConnectionManager().GetConnection(checker.Ballot.Proposer())
	if client == nil {
		err = errors.ErrorBallotFromUnknownValidator
		return
	}
	var body []byte
	// TODO check error
	if body, err = client.GetTransactions(unknown); err != nil {
		return
	}

	var receivedTransaction []transaction.Transaction
	bf := bufio.NewReader(bytes.NewReader(body))
	for {
		var l []byte
		l, err = bf.ReadBytes('\n')
		if err == io.EOF {
			err = nil
			break
		} else if err != nil {
			return
		}
		var itemType NodeItemDataType
		var d interface{}
		if itemType, d, err = UnmarshalNodeItemResponse(l); err != nil {
			return
		}
		if itemType == NodeItemError {
			err = d.(*errors.Error)
			return
		}

		var tx transaction.Transaction
		var ok bool
		if tx, ok = d.(transaction.Transaction); !ok {
			err = errors.ErrorTransactionNotFound
			return
		}
		if err = tx.IsWellFormed(checker.NetworkID, checker.NodeRunner.Conf); err != nil {
			return
		}

		receivedTransaction = append(receivedTransaction, tx)
	}

	for _, tx := range receivedTransaction {
		checker.NodeRunner.TransactionPool.Add(tx)
	}

	return
}

func BallotGetMissingTransaction(c common.Checker, args ...interface{}) (err error) {
	checker := c.(*BallotChecker)

	if checker.VotingHole != ballot.VotingNOTYET {
		return
	}

	if err = getMissingTransaction(checker); err != nil {
		checker.VotingHole = ballot.VotingNO
		err = nil
		checker.Log.Debug("failed to get the missing transactions of ballot", "error", err)
	}

	return
}

var INITBallotTransactionCheckerFuncs = []common.CheckerFunc{
	IsNew,
	CheckMissingTransaction,
	BallotTransactionsSameSource,
	BallotTransactionsSourceCheck,
	BallotTransactionsOperationBodyCollectTxFee,
	BallotTransactionsAllValid,
}

// INITBallotValidateTransactions validates the
// transactions of newly added ballot.
func INITBallotValidateTransactions(c common.Checker, args ...interface{}) (err error) {
	checker := c.(*BallotChecker)

	if checker.VotingFinished {
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

	transactionsChecker := &BallotTransactionChecker{
		DefaultChecker: common.DefaultChecker{Funcs: INITBallotTransactionCheckerFuncs},
		NodeRunner:     checker.NodeRunner,
		LocalNode:      checker.LocalNode,
		NetworkID:      checker.NetworkID,
		Ballot:         checker.Ballot,
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

	newBallot := checker.Ballot
	newBallot.SetSource(checker.LocalNode.Address())
	newBallot.SetVote(ballot.StateSIGN, checker.VotingHole)
	newBallot.Sign(checker.LocalNode.Keypair(), checker.NetworkID)

	if !checker.NodeRunner.Consensus().HasRunningRound(checker.Ballot.VotingBasis().Index()) {
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
	checker.NodeRunner.TransitISAACState(checker.Ballot.VotingBasis(), ballot.StateSIGN)

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

	if !checker.NodeRunner.Consensus().HasRunningRound(checker.Ballot.VotingBasis().Index()) {
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
	checker.NodeRunner.TransitISAACState(checker.Ballot.VotingBasis(), ballot.StateACCEPT)

	return
}

// FinishedBallotStore will store the confirmed ballot to
// `Block`.
func FinishedBallotStore(c common.Checker, args ...interface{}) (err error) {
	checker := c.(*BallotChecker)

	if !checker.VotingFinished {
		return
	}
	ballotRound := checker.Ballot.VotingBasis()
	if checker.FinishedVotingHole == ballot.VotingYES {
		if err = getMissingTransaction(checker); err != nil {
			checker.Log.Debug("failed to get the missing transactions of ballot", "error", err)
			return
		}

		var theBlock *block.Block

		var bs *storage.LevelDBBackend
		if bs, err = checker.NodeRunner.Storage().OpenBatch(); err != nil {
			return err
		}

		theBlock, err = finishBallot(
			bs,
			checker.Ballot,
			checker.NodeRunner.TransactionPool,
			checker.Log,
			checker.NodeRunner.Log(),
		)
		if err != nil {
			bs.Discard()
			checker.Log.Error("failed to finish ballot", "error", err)
			return
		}

		if err = bs.Commit(); err != nil {
			if err != errors.ErrorNotCommittableBackend {
				bs.Discard()
				return
			}
		}

		checker.Log.Debug("ballot was stored", "block", *theBlock)
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
		checker.NodeRunner.TransactionPool,
	)

	return
}

func finishBallot(st *storage.LevelDBBackend, b ballot.Ballot, transactionPool *transaction.Pool, log, infoLog logging.Logger) (*block.Block, error) {
	var err error
	var isValid bool
	if isValid, err = isValidRound(st, b.VotingBasis(), infoLog); err != nil || !isValid {
		return nil, err
	}

	var nOps int
	for _, hash := range b.B.Proposed.Transactions {
		tx, found := transactionPool.Get(hash)
		if !found {
			return nil, errors.ErrorTransactionNotFound
		}
		nOps += len(tx.B.Operations)
	}

	r := b.VotingBasis()
	r.Height++                                      // next block
	r.TotalTxs += uint64(len(b.Transactions()) + 1) // + 1 for ProposerTransaction
	r.TotalOps += uint64(nOps + len(b.ProposerTransaction().B.Operations))

	blk := block.NewBlock(
		b.Proposer(),
		r,
		b.ProposerTransaction().GetHash(),
		b.Transactions(),
		b.ProposerConfirmed(),
	)

	if err = blk.Save(st); err != nil {
		log.Error("failed to create new block", "block", blk, "error", err)
		return nil, err
	}

	log.Debug("NewBlock created", "block", blk)
	infoLog.Info("NewBlock created",
		"height", blk.Height,
		"round", blk.Round,
		"timestamp", blk.Timestamp,
		"total-txs", blk.TotalTxs,
		"total-ops", blk.TotalOps,
		"proposer", blk.Proposer,
	)

	pTxHashes := b.B.Proposed.Transactions
	proposedTransactions := make([]*transaction.Transaction, 0, len(pTxHashes))
	for _, hash := range pTxHashes {
		tx, found := transactionPool.Get(hash)
		if !found {
			err = errors.ErrorTransactionNotFound
			return nil, err
		}
		proposedTransactions = append(proposedTransactions, &tx)
	}

	if err = FinishTransactions(*blk, proposedTransactions, st); err != nil {
		return nil, err
	}

	if err = FinishProposerTransaction(st, *blk, b.ProposerTransaction(), log); err != nil {
		log.Error("failed to finish proposer transaction", "block", blk, "ptx", b.ProposerTransaction(), "error", err)
		return nil, err
	}

	return blk, nil
}

func isValidRound(st *storage.LevelDBBackend, r voting.Basis, log logging.Logger) (bool, error) {
	latestBlock := block.GetLatestBlock(st)
	if latestBlock.Height != r.Height {
		log.Error(
			"ballot height is not equal to latestBlock",
			"in ballot", r.Height,
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

func FinishTransactions(blk block.Block, transactions []*transaction.Transaction, st *storage.LevelDBBackend) (err error) {
	for _, tx := range transactions {
		raw, _ := json.Marshal(tx)

		bt := block.NewBlockTransactionFromTransaction(blk.Hash, blk.Height, blk.Confirmed, *tx, raw)
		if err = bt.Save(st); err != nil {
			return
		}
		for _, op := range tx.B.Operations {
			if err = finishOperation(st, tx.B.Source, op, log); err != nil {
				log.Error("failed to finish operation", "block", blk, "bt", bt, "op", op, "error", err)
				return err
			}
		}

		var baSource *block.BlockAccount
		if baSource, err = block.GetBlockAccount(st, tx.B.Source); err != nil {
			err = errors.ErrorBlockAccountDoesNotExists
			return
		}

		if err = baSource.Withdraw(tx.TotalAmount(true)); err != nil {
			return
		}

		if err = baSource.Save(st); err != nil {
			return
		}

	}
	return
}

// finishOperation do finish the task after consensus by the type of each operation.
func finishOperation(st *storage.LevelDBBackend, source string, op operation.Operation, log logging.Logger) (err error) {
	switch op.H.Type {
	case operation.TypeCreateAccount:
		pop, ok := op.B.(operation.CreateAccount)
		if !ok {
			return errors.ErrorUnknownOperationType
		}
		return finishCreateAccount(st, source, pop, log)
	case operation.TypePayment:
		pop, ok := op.B.(operation.Payment)
		if !ok {
			return errors.ErrorUnknownOperationType
		}
		return finishPayment(st, source, pop, log)
	case operation.TypeCongressVoting, operation.TypeCongressVotingResult:
		//Nothing to do
		return
	case operation.TypeUnfreezingRequest:
		pop, ok := op.B.(operation.UnfreezeRequest)
		if !ok {
			return errors.ErrorUnknownOperationType
		}
		return finishUnfreezeRequest(st, source, pop, log)
	default:
		err = errors.ErrorUnknownOperationType
		return
	}
}

func finishCreateAccount(st *storage.LevelDBBackend, source string, op operation.CreateAccount, log logging.Logger) (err error) {

	var baSource, baTarget *block.BlockAccount
	if baSource, err = block.GetBlockAccount(st, source); err != nil {
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

func finishPayment(st *storage.LevelDBBackend, source string, op operation.Payment, log logging.Logger) (err error) {

	var baSource, baTarget *block.BlockAccount
	if baSource, err = block.GetBlockAccount(st, source); err != nil {
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

func FinishProposerTransaction(st *storage.LevelDBBackend, blk block.Block, ptx ballot.ProposerTransaction, log logging.Logger) (err error) {
	{
		var opb operation.CollectTxFee
		if opb, err = ptx.CollectTxFee(); err != nil {
			return
		}
		if err = finishCollectTxFee(st, opb, log); err != nil {
			return
		}
	}

	{
		var opb operation.Inflation
		if opb, err = ptx.Inflation(); err != nil {
			return
		}
		if err = finishInflation(st, opb, log); err != nil {
			return
		}
	}

	raw, _ := json.Marshal(ptx.Transaction)
	bt := block.NewBlockTransactionFromTransaction(blk.Hash, blk.Height, blk.Confirmed, ptx.Transaction, raw)
	if err = bt.Save(st); err != nil {
		return
	}

	return
}

func finishCollectTxFee(st *storage.LevelDBBackend, opb operation.CollectTxFee, log logging.Logger) (err error) {
	if opb.Amount < 1 {
		return
	}

	var commonAccount *block.BlockAccount
	if commonAccount, err = block.GetBlockAccount(st, opb.TargetAddress()); err != nil {
		return
	}

	if err = commonAccount.Deposit(opb.GetAmount()); err != nil {
		return
	}

	if err = commonAccount.Save(st); err != nil {
		return
	}

	return
}

func finishInflation(st *storage.LevelDBBackend, opb operation.Inflation, log logging.Logger) (err error) {
	if opb.Amount < 1 {
		return
	}

	var commonAccount *block.BlockAccount
	if commonAccount, err = block.GetBlockAccount(st, opb.TargetAddress()); err != nil {
		return
	}

	if err = commonAccount.Deposit(opb.GetAmount()); err != nil {
		return
	}

	if err = commonAccount.Save(st); err != nil {
		return
	}

	return
}

func finishUnfreezeRequest(st *storage.LevelDBBackend, source string, opb operation.UnfreezeRequest, log logging.Logger) (err error) {

	log.Debug("UnfreezeRequest done")

	return
}
