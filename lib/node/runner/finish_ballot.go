package runner

import (
	logging "github.com/inconshreveable/log15"

	"boscoin.io/sebak/lib/ballot"
	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/errors"
	"boscoin.io/sebak/lib/storage"
	"boscoin.io/sebak/lib/transaction"
	"boscoin.io/sebak/lib/transaction/operation"
)

func finishBallot(st *storage.LevelDBBackend, b ballot.Ballot, transactionPool *transaction.Pool, log, infoLog logging.Logger) (*block.Block, error) {
	proposedTxs, err := getProposedTransactions(
		st,
		b.B.Proposed.Transactions,
		transactionPool,
	)
	if err != nil {
		return nil, err
	}

	var blk *block.Block
	blk, err = finishBallotWithProposedTxs(
		st,
		b,
		transactionPool.Pending,
		proposedTxs,
		log,
		infoLog,
	)
	if err != nil {
		return nil, err
	}

	return blk, nil
}

func finishBallotWithProposedTxs(st *storage.LevelDBBackend, b ballot.Ballot, pool *transaction.PendingPool, proposedTransactions []*transaction.Transaction, log, infoLog logging.Logger) (*block.Block, error) {
	var err error
	var isValid bool
	if isValid, err = isValidRound(st, b.VotingBasis(), infoLog); err != nil || !isValid {
		return nil, err
	}

	var nOps int
	for _, tx := range proposedTransactions {
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

	if err = FinishTransactions(*blk, proposedTransactions, pool, st); err != nil {
		return nil, err
	}

	if err = FinishProposerTransaction(st, *blk, b.ProposerTransaction(), log); err != nil {
		log.Error("failed to finish proposer transaction", "block", blk, "ptx", b.ProposerTransaction(), "error", err)
		return nil, err
	}

	if err = finishDelayedOperations(blk, pool, st); err != nil {
		log.Error("Applying delayed transactions failed", "block", blk)
		return nil, err
	}

	return blk, nil
}

func getProposedTransactions(st *storage.LevelDBBackend, pTxHashes []string, transactionPool *transaction.Pool) ([]*transaction.Transaction, error) {
	proposedTransactions := make([]*transaction.Transaction, 0, len(pTxHashes))
	var err error
	for _, hash := range pTxHashes {
		tx, found := transactionPool.Get(hash)
		if !found {
			var tp block.TransactionPool
			if tp, err = block.GetTransactionPool(st, hash); err != nil {
				return nil, errors.TransactionNotFound
			}
			tx = tp.Transaction()
		}
		proposedTransactions = append(proposedTransactions, &tx)
	}
	return proposedTransactions, nil
}

func FinishTransactions(blk block.Block, transactions []*transaction.Transaction, pool *transaction.PendingPool, st *storage.LevelDBBackend) (err error) {
	for _, tx := range transactions {
		bt := block.NewBlockTransactionFromTransaction(blk.Hash, blk.Height, blk.Confirmed, *tx)
		if err = bt.Save(st); err != nil {
			return
		}
		for _, op := range tx.B.Operations {
			opKey := block.NewBlockOperationKey(op.MakeHashString(), tx.GetHash())
			if err = finishOperation(st, pool, tx.B.Source, op, opKey, log); err != nil {
				log.Error("failed to finish operation", "block", blk, "bt", bt, "op", op, "error", err)
				return err
			}
		}

		var baSource *block.BlockAccount
		if baSource, err = block.GetBlockAccount(st, tx.B.Source); err != nil {
			err = errors.BlockAccountDoesNotExists
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
func finishOperation(st *storage.LevelDBBackend, pool *transaction.PendingPool, source string, op operation.Operation, opKey string, log logging.Logger) (err error) {
	switch op.H.Type {
	case operation.TypeCreateAccount:
		pop, ok := op.B.(operation.CreateAccount)
		if !ok {
			return errors.UnknownOperationType
		}
		return finishCreateAccount(st, source, pop, log)
	case operation.TypePayment:
		pop, ok := op.B.(operation.Payment)
		if !ok {
			return errors.UnknownOperationType
		}
		return finishPayment(st, pool, source, opKey, pop, log)
	case operation.TypeCongressVoting, operation.TypeCongressVotingResult:
		//Nothing to do
		return
	case operation.TypeUnfreezingRequest:
		pop, ok := op.B.(operation.UnfreezeRequest)
		if !ok {
			return errors.UnknownOperationType
		}
		return finishUnfreezeRequest(st, source, pop, log)
	default:
		err = errors.UnknownOperationType
		return
	}
}

func finishCreateAccount(st *storage.LevelDBBackend, source string, op operation.CreateAccount, log logging.Logger) (err error) {
	if _, err = block.GetBlockAccount(st, source); err != nil {
		err = errors.BlockAccountDoesNotExists
		return
	}

	var baTarget *block.BlockAccount
	if baTarget, err = block.GetBlockAccount(st, op.TargetAddress()); err == nil {
		err = errors.BlockAccountAlreadyExists
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

	return
}

//
// Apply delayed operations for this block height
//
// If any operation is pending for this height, it will be applied by this function.
// Since delayed operations are only added via previous operations,
// the content of this pool is guaranteed to be in sync for any well-behaved node.
//
// Params:
//   blk  = Pointer to the Block being finalized
//   pool = The PendingPool to get operations from
//   st   = Storage, used to look up the required blocks (and apply the changes)
//
// Returns:
//   `nil` on success, in which case `pending` might have been mutated.
//   If the return is non-`nil`, pending won't have been mutated.
func finishDelayedOperations(blk *block.Block, pending *transaction.PendingPool, st *storage.LevelDBBackend) error {
	var err error
	offset := uint64(0)
	for pendingKey := pending.Peek(blk.Height, 0); pendingKey != ""; pendingKey = pending.Peek(blk.Height, offset) {
		var bop block.BlockOperation
		var btx block.BlockTransaction
		if bop, err = block.GetBlockOperation(st, pendingKey); err == nil {
			if btx, err = block.GetBlockTransaction(st, bop.TxHash); err == nil {
				var op_body operation.Body
				if op_body, err = operation.UnmarshalBodyJSON(bop.Type, bop.Body); err == nil {
					op := operation.Operation{
						H: operation.Header{Type: bop.Type},
						B: op_body,
					}
					err = finishOperation(st, nil, btx.Source, op, "", log)
				}
			}
		}
		if err != nil {
			log.Error("applying delayed operation failed", "block", *blk, "btx", btx, "BlockOperation", bop, "pending", pendingKey, "error", err)
			return err
		}
		offset += 1
	}
	// Done outside of the loop so that any failure is does not lead to records being removed
	pending.PopHeight(blk.Height)
	return nil
}

func finishPayment(st *storage.LevelDBBackend, pool *transaction.PendingPool, source string, opKey string, op operation.Payment, log logging.Logger) (err error) {
	var baSource, baTarget *block.BlockAccount
	if baSource, err = block.GetBlockAccount(st, source); err != nil {
		err = errors.BlockAccountDoesNotExists
		return
	}

	if baTarget, err = block.GetBlockAccount(st, op.TargetAddress()); err != nil {
		err = errors.BlockAccountDoesNotExists
		return
	}

	// Store unfreezing request
	if baSource.Linked != "" && pool != nil {
		target := block.GetLatestBlock(st).Height + common.UnfreezingPeriod
		pool.Insert(target, opKey)
		return nil
	}

	if err = baTarget.Deposit(op.GetAmount()); err != nil {
		return
	}
	if err = baTarget.Save(st); err != nil {
		return
	}

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

	bt := block.NewBlockTransactionFromTransaction(blk.Hash, blk.Height, blk.Confirmed, ptx.Transaction)
	if err = bt.Save(st); err != nil {
		return
	}

	if _, err = block.SaveTransactionPool(st, ptx.Transaction); err != nil {
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
	return
}
