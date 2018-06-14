package sebak

import (
	"fmt"

	"github.com/btcsuite/btcutil/base58"
	"github.com/stellar/go/keypair"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/error"
)

type TransactionChecker struct {
	sebakcommon.DefaultChecker

	NetworkID   []byte
	Transaction Transaction
}

func CheckTransactionSource(c sebakcommon.Checker, args ...interface{}) (err error) {
	checker := c.(*TransactionChecker)
	if _, err = keypair.Parse(checker.Transaction.B.Source); err != nil {
		err = sebakerror.ErrorBadPublicAddress
		return
	}

	return
}

func CheckTransactionBaseFee(c sebakcommon.Checker, args ...interface{}) (err error) {
	checker := c.(*TransactionChecker)
	if checker.Transaction.B.Fee < BaseFee {
		err = sebakerror.ErrorInvalidFee
		return
	}

	return
}

func CheckTransactionOperation(c sebakcommon.Checker, args ...interface{}) (err error) {
	checker := c.(*TransactionChecker)

	var hashes []string
	for _, op := range checker.Transaction.B.Operations {
		if checker.Transaction.B.Source == op.B.TargetAddress() {
			err = sebakerror.ErrorInvalidOperation
			return
		}
		if err = op.IsWellFormed(checker.NetworkID); err != nil {
			return
		}
		// if there are multiple operations which has same 'Type' and same
		// 'TargetAddress()', this transaction will be invalid.
		u := fmt.Sprintf("%s-%s", op.H.Type, op.B.TargetAddress())
		if _, found := sebakcommon.InStringArray(hashes, u); found {
			err = sebakerror.ErrorDuplicatedOperation
			return
		}

		hashes = append(hashes, u)
	}

	return
}

func CheckTransactionVerifySignature(c sebakcommon.Checker, args ...interface{}) (err error) {
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

func CheckTransactionHashMatch(c sebakcommon.Checker, args ...interface{}) (err error) {
	checker := c.(*TransactionChecker)
	if checker.Transaction.H.Hash != checker.Transaction.B.MakeHashString() {
		err = sebakerror.ErrorHashDoesNotMatch
		return
	}

	return
}
