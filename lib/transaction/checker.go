package transaction

import (
	"fmt"

	"github.com/btcsuite/btcutil/base58"
	"github.com/stellar/go/keypair"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/error"
	"boscoin.io/sebak/lib/transaction/operation"
)

type Checker struct {
	common.DefaultChecker

	NetworkID       []byte
	Transaction     Transaction
	OperationsLimit int
}

func CheckSource(c common.Checker, args ...interface{}) (err error) {
	checker := c.(*Checker)
	if _, err = keypair.Parse(checker.Transaction.B.Source); err != nil {
		err = errors.ErrorBadPublicAddress
		return
	}

	return
}

func CheckOverOperationsLimit(c common.Checker, args ...interface{}) (err error) {
	checker := c.(*Checker)

	if len(checker.Transaction.B.Operations) > operation.Limit {
		err = errors.ErrorTransactionHasOverMaxOperations
		return
	}

	return
}

func CheckSequenceID(c common.Checker, args ...interface{}) (err error) {
	//checker := c.(*Checker)
	return
}

func CheckBaseFee(c common.Checker, args ...interface{}) (err error) {
	checker := c.(*Checker)
	if checker.Transaction.B.Fee < checker.Transaction.TotalBaseFee() {
		err = errors.ErrorInvalidFee
		return
	}

	return
}

func CheckOperationTypes(c common.Checker, args ...interface{}) (err error) {
	checker := c.(*Checker)

	if len(checker.Transaction.B.Operations) < 1 {
		err = errors.ErrorTransactionEmptyOperations
		return
	}

	for _, op := range checker.Transaction.B.Operations {
		if _, found := operation.KindsNormalTransaction[op.H.Type]; !found {
			err = errors.ErrorInvalidOperation
			return
		}
	}

	return
}

func CheckOperations(c common.Checker, args ...interface{}) (err error) {
	checker := c.(*Checker)

	var hashes []string
	for _, op := range checker.Transaction.B.Operations {
		if pop, ok := op.B.(operation.Payable); ok {
			if checker.Transaction.B.Source == pop.TargetAddress() {
				err = errors.ErrorInvalidOperation
				return
			}
			if err = op.IsWellFormed(checker.NetworkID); err != nil {
				return
			}
			// if there are multiple operations which has same 'Type' and same
			// 'TargetAddress()', this transaction will be invalid.
			u := fmt.Sprintf("%s-%s", op.H.Type, pop.TargetAddress())
			if _, found := common.InStringArray(hashes, u); found {
				err = errors.ErrorDuplicatedOperation
				return
			}

			hashes = append(hashes, u)
		}
	}

	return
}

func CheckVerifySignature(c common.Checker, args ...interface{}) (err error) {
	checker := c.(*Checker)

	var kp keypair.KP
	if kp, err = keypair.Parse(checker.Transaction.B.Source); err != nil {
		return
	}
	err = kp.Verify(
		append(checker.NetworkID, []byte(checker.Transaction.H.Hash)...),
		base58.Decode(checker.Transaction.H.Signature),
	)
	if err != nil {
		return
	}
	return
}
