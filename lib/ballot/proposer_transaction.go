package ballot

import (
	"encoding/json"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/error"
	"boscoin.io/sebak/lib/transaction"
	"boscoin.io/sebak/lib/transaction/operation"
)

var TypesProposerTransaction map[operation.OperationType]struct{} = map[operation.OperationType]struct{}{
	operation.TypeCollectTxFee: struct{}{},
	operation.TypeInflation:    struct{}{},
}

type ProposerTransaction struct {
	transaction.Transaction
}

func NewProposerTransaction(proposer string, ops ...operation.Operation) (ptx ProposerTransaction, err error) {
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

func NewCollectTxFeeFromBallot(blt Ballot, commonAccount string, txs ...transaction.Transaction) (opb operation.CollectTxFee, err error) {
	rd := blt.Round()

	var feeAmount common.Amount
	for _, tx := range txs {
		feeAmount = feeAmount + tx.B.Fee
	}

	opb = operation.NewCollectTxFee(
		commonAccount,
		feeAmount,
		uint64(len(txs)),
		rd.BlockHeight,
		rd.BlockHash,
		rd.TotalTxs,
	)
	return
}

func NewInflationFromBallot(blt Ballot, commonAccount string, initialBalance common.Amount) (opb operation.Inflation, err error) {
	rd := blt.Round()

	var amount common.Amount
	if amount, err = common.CalculateInflation(initialBalance); err != nil {
		return
	}

	opb = operation.NewOperationBodyInflation(
		commonAccount,
		amount,
		initialBalance,
		rd.BlockHeight,
		rd.BlockHash,
		rd.TotalTxs,
	)

	return
}

func NewProposerTransactionFromBallot(blt Ballot, opc operation.CollectTxFee, opi operation.Inflation) (ptx ProposerTransaction, err error) {
	var ops []operation.Operation

	var op operation.Operation
	{ // OperationCollectTxFee
		if op, err = operation.NewOperation(opc); err != nil {
			return
		}
		ops = append(ops, op)
	}

	{ // OperationInflation
		if op, err = operation.NewOperation(opi); err != nil {
			return
		}
		ops = append(ops, op)
	}

	ptx, err = NewProposerTransaction(blt.Proposer(), ops...)

	return
}

var ProposerTransactionWellFormedCheckerFuncs = []common.CheckerFunc{
	transaction.CheckOverOperationsLimit,
	transaction.CheckSequenceID,
	transaction.CheckSource,
	CheckProposerTransactionFee,
	CheckProposerTransactionOperationTypes,
	transaction.CheckOperations,
	transaction.CheckVerifySignature,
}

func (p ProposerTransaction) IsWellFormed(networkID []byte, conf common.Config) (err error) {
	if _, err = p.CollectTxFee(); err != nil {
		return
	}

	checker := &transaction.Checker{
		DefaultChecker: common.DefaultChecker{Funcs: ProposerTransactionWellFormedCheckerFuncs},
		NetworkID:      networkID,
		Transaction:    p.Transaction,
		Conf:           conf,
	}
	if err = common.RunChecker(checker, common.DefaultDeferFunc); err != nil {
		return
	}

	return
}

func (p ProposerTransaction) IsWellFormedWithBallot(networkID []byte, blt Ballot, conf common.Config) (err error) {
	if p.Source() != blt.Proposer() {
		err = errors.ErrorInvalidProposerTransaction
		return
	}

	if err = p.IsWellFormed(networkID, conf); err != nil {
		return
	}

	rd := blt.Round()
	{ // check OperationCollectTxFee
		var opb operation.CollectTxFee
		if opb, err = blt.ProposerTransaction().CollectTxFee(); err != nil {
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
		var opb operation.Inflation
		if opb, err = blt.ProposerTransaction().Inflation(); err != nil {
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

func (p ProposerTransaction) CollectTxFee() (opb operation.CollectTxFee, err error) {
	var found bool
	for _, op := range p.B.Operations {
		switch op.B.(type) {
		case operation.CollectTxFee:
			opb = op.B.(operation.CollectTxFee)
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

func (p ProposerTransaction) Inflation() (opb operation.Inflation, err error) {
	var found bool
	for _, op := range p.B.Operations {
		switch op.B.(type) {
		case operation.Inflation:
			opb = op.B.(operation.Inflation)
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
	checker := c.(*transaction.Checker)
	if checker.Transaction.B.Fee != 0 {
		err = errors.ErrorInvalidFee
		return
	}

	return
}

func CheckProposerTransactionOperationTypes(c common.Checker, args ...interface{}) (err error) {
	checker := c.(*transaction.Checker)

	if len(checker.Transaction.B.Operations) != 2 {
		err = errors.ErrorInvalidProposerTransaction
		return
	}

	var foundTypes []string
	for _, op := range checker.Transaction.B.Operations {
		if _, found := TypesProposerTransaction[op.H.Type]; !found {
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
