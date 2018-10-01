package ballot

import (
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/error"
	"boscoin.io/sebak/lib/transaction"
)

var availableOperationTypesProposerTransaction map[transaction.OperationType]struct{} = map[transaction.OperationType]struct{}{
	transaction.OperationCollectTxFee: struct{}{},
}

type ProposerTransaction struct {
	transaction.Transaction
}

func NewProposerTransaction(proposer string, op transaction.Operation) (ptx ProposerTransaction, err error) {
	var tx transaction.Transaction
	tx, err = transaction.NewTransaction(proposer, 0, op)
	if err != nil {
		return
	}
	tx.B.Fee = 0

	ptx = ProposerTransaction{Transaction: tx}

	return
}

func NewProposerTransactionFromBallot(blt Ballot, commonAccount string, txs ...transaction.Transaction) (ptx ProposerTransaction, err error) {
	var feeAmount common.Amount
	for _, tx := range txs {
		feeAmount = feeAmount + tx.B.Fee
	}

	rd := blt.Round()
	opb := transaction.NewOperationBodyCollectTxFee(
		commonAccount,
		feeAmount,
		uint64(len(txs)),
		rd.BlockHeight,
		rd.BlockHash,
		rd.TotalTxs,
	)

	var op transaction.Operation
	if op, err = transaction.NewOperation(opb); err != nil {
		return
	}

	ptx, err = NewProposerTransaction(blt.Proposer(), op)

	return
}

var ProposerTransactionWellFormedCheckerFuncs = []common.CheckerFunc{
	transaction.CheckTransactionOverOperationsLimit,
	transaction.CheckTransactionSequenceID,
	transaction.CheckTransactionSource,
	CheckProposerTransactionFee,
	transaction.CheckTransactionOperation,
	transaction.CheckTransactionVerifySignature,
}

func (p ProposerTransaction) IsWellFormed(networkID []byte) (err error) {
	if _, err = p.OperationBodyCollectTxFee(); err != nil {
		return
	}

	checker := &transaction.TransactionChecker{
		DefaultChecker: common.DefaultChecker{Funcs: ProposerTransactionWellFormedCheckerFuncs},
		NetworkID:      networkID,
		Transaction:    p.Transaction,
	}
	if err = common.RunChecker(checker, common.DefaultDeferFunc); err != nil {
		return
	}

	return
}

func (p ProposerTransaction) IsWellFormedWithBallot(networkID []byte, blt Ballot) (err error) {
	if p.Source() != blt.Proposer() {
		err = errors.ErrorInvalidProposerTransaction
		return
	}

	if err = p.IsWellFormed(networkID); err != nil {
		return
	}

	// check the expected collected Fee is right
	var opb transaction.OperationBodyCollectTxFee
	if opb, err = blt.ProposerTransaction().OperationBodyCollectTxFee(); err != nil {
		return
	}

	if opb.Txs != uint64(blt.TransactionsLength()) {
		err = errors.ErrorInvalidOperation
		return
	}

	rd := blt.Round()
	if opb.BlockHeight != rd.BlockHeight {
		err = errors.ErrorInvalidOperation
		return
	}
	if opb.BlockHash != rd.BlockHash {
		err = errors.ErrorInvalidOperation
		return
	}
	if opb.TotalTxs != rd.TotalTxs {
		err = errors.ErrorInvalidOperation
		return
	}

	if len(blt.Transactions()) < 1 {
		if opb.Amount != 0 {
			err = errors.ErrorInvalidOperation
			return
		}
	} else if opb.Amount < 1 {
		err = errors.ErrorInvalidOperation
		return
	}

	return
}

func (p ProposerTransaction) OperationBodyCollectTxFee() (opb transaction.OperationBodyCollectTxFee, err error) {
	var found bool
	for _, op := range p.B.Operations {
		switch op.B.(type) {
		case transaction.OperationBodyCollectTxFee:
			opb = op.B.(transaction.OperationBodyCollectTxFee)
			found = true
		default:
			continue
		}
	}

	if !found {
		err = errors.ErrorInvalidProposerTransaction
		return
	}

	return
}

func CheckProposerTransactionFee(c common.Checker, args ...interface{}) (err error) {
	checker := c.(*transaction.TransactionChecker)
	if checker.Transaction.B.Fee != 0 {
		err = errors.ErrorInvalidFee
		return
	}

	return
}
