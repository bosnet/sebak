package ballot

import (
	"encoding/json"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/error"
	"boscoin.io/sebak/lib/transaction"
)

var OperationTypesProposerTransaction map[transaction.OperationType]struct{} = map[transaction.OperationType]struct{}{
	transaction.OperationCollectTxFee: struct{}{},
	transaction.OperationInflation:    struct{}{},
}

type ProposerTransaction struct {
	transaction.Transaction
}

func NewProposerTransaction(proposer string, ops ...transaction.Operation) (ptx ProposerTransaction, err error) {
	var tx transaction.Transaction
	tx, err = transaction.NewTransaction(proposer, 0, ops...)
	if err != nil {
		return
	}
	tx.B.Fee = 0
	tx.H.Hash = tx.B.MakeHashString()

	ptx = ProposerTransaction{Transaction: tx}

	return
}

func NewOperationCollectTxFeeFromBallot(blt Ballot, commonAccount string, txs ...transaction.Transaction) (opb transaction.OperationBodyCollectTxFee, err error) {
	rd := blt.Round()

	var feeAmount common.Amount
	for _, tx := range txs {
		feeAmount = feeAmount + tx.B.Fee
	}

	opb = transaction.NewOperationBodyCollectTxFee(
		commonAccount,
		feeAmount,
		uint64(len(txs)),
		rd.BlockHeight,
		rd.BlockHash,
		rd.TotalTxs,
	)
	return
}

func NewOperationInflationFromBallot(blt Ballot, commonAccount string, initialBalance common.Amount, ratio float64) (opb transaction.OperationBodyInflation, err error) {
	rd := blt.Round()

	var amount common.Amount
	if amount, err = common.CalculateInflation(initialBalance, ratio); err != nil {
		return
	}

	opb = transaction.NewOperationBodyInflation(
		commonAccount,
		amount,
		initialBalance,
		ratio,
		rd.BlockHeight,
		rd.BlockHash,
		rd.TotalTxs,
	)

	return
}

func NewProposerTransactionFromBallot(blt Ballot, opc transaction.OperationBodyCollectTxFee, opi transaction.OperationBodyInflation) (ptx ProposerTransaction, err error) {
	var ops []transaction.Operation

	var op transaction.Operation
	{ // OperationCollectTxFee
		if op, err = transaction.NewOperation(opc); err != nil {
			return
		}
		ops = append(ops, op)
	}

	{ // OperationInflation
		if op, err = transaction.NewOperation(opi); err != nil {
			return
		}
		ops = append(ops, op)
	}

	ptx, err = NewProposerTransaction(blt.Proposer(), ops...)

	return
}

var ProposerTransactionWellFormedCheckerFuncs = []common.CheckerFunc{
	transaction.CheckTransactionOverOperationsLimit,
	transaction.CheckTransactionSequenceID,
	transaction.CheckTransactionSource,
	CheckProposerTransactionFee,
	CheckProposerTransactionOperationTypes,
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

	rd := blt.Round()
	{ // check OperationCollectTxFee
		var opb transaction.OperationBodyCollectTxFee
		if opb, err = blt.ProposerTransaction().OperationBodyCollectTxFee(); err != nil {
			return
		}

		if opb.Txs != uint64(blt.TransactionsLength()) {
			err = errors.ErrorInvalidOperation
			return
		}

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
	}

	{ // check OperationInflation
		var opb transaction.OperationBodyInflation
		if opb, err = blt.ProposerTransaction().OperationBodyInflation(); err != nil {
			return
		}

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
			break
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

func (p ProposerTransaction) OperationBodyInflation() (opb transaction.OperationBodyInflation, err error) {
	var found bool
	for _, op := range p.B.Operations {
		switch op.B.(type) {
		case transaction.OperationBodyInflation:
			opb = op.B.(transaction.OperationBodyInflation)
			found = true
			break
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

func (p *ProposerTransaction) UnmarshalJSON(b []byte) error {
	var t transaction.Transaction
	if err := json.Unmarshal(b, &t); err != nil {
		return err
	}
	t.H.Hash = t.B.MakeHashString()

	*p = ProposerTransaction{Transaction: t}

	return nil
}

func CheckProposerTransactionFee(c common.Checker, args ...interface{}) (err error) {
	checker := c.(*transaction.TransactionChecker)
	if checker.Transaction.B.Fee != 0 {
		err = errors.ErrorInvalidFee
		return
	}

	return
}

func CheckProposerTransactionOperationTypes(c common.Checker, args ...interface{}) (err error) {
	checker := c.(*transaction.TransactionChecker)

	if len(checker.Transaction.B.Operations) != 2 {
		err = errors.ErrorInvalidProposerTransaction
		return
	}

	var foundTypes []string
	for _, op := range checker.Transaction.B.Operations {
		if _, found := OperationTypesProposerTransaction[op.H.Type]; !found {
			err = errors.ErrorInvalidOperation
			return
		}
		if _, found := common.InStringArray(foundTypes, string(op.H.Type)); found {
			err = errors.ErrorDuplicatedOperation
			return
		}

		foundTypes = append(foundTypes, string(op.H.Type))
	}

	return
}
