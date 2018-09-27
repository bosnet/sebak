package transaction

import (
	"fmt"

	"github.com/btcsuite/btcutil/base58"
	"github.com/stellar/go/keypair"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/error"
)

type TransactionChecker struct {
	common.DefaultChecker

	NetworkID   []byte
	Transaction Transaction
}

func CheckTransactionSource(c common.Checker, args ...interface{}) (err error) {
	checker := c.(*TransactionChecker)
	if _, err = keypair.Parse(checker.Transaction.B.Source); err != nil {
		err = errors.ErrorBadPublicAddress
		return
	}

	return
}

func CheckTransactionOverOperationsLimit(c common.Checker, args ...interface{}) (err error) {
	checker := c.(*TransactionChecker)

	if len(checker.Transaction.B.Operations) > common.MaxOperationsInTransaction {
		err = errors.ErrorTransactionHasOverMaxOperations
		return
	}

	return
}

func CheckTransactionSequenceID(c common.Checker, args ...interface{}) (err error) {
	//checker := c.(*TransactionChecker)
	return
}

func CheckTransactionBaseFee(c common.Checker, args ...interface{}) (err error) {
	checker := c.(*TransactionChecker)
	if checker.Transaction.B.Fee < common.BaseFee {
		err = errors.ErrorInvalidFee
		return
	}

	return
}

func CheckTransactionOperation(c common.Checker, args ...interface{}) (err error) {
	checker := c.(*TransactionChecker)

	if len(checker.Transaction.B.Operations) < 1 {
		err = errors.ErrorTransactionEmptyOperations
		return
	}

	var hashes []string
	for _, op := range checker.Transaction.B.Operations {
		if pop, ok := op.B.(OperationBodyPayable); ok {
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

func CheckTransactionVerifySignature(c common.Checker, args ...interface{}) (err error) {
	checker := c.(*TransactionChecker)

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
