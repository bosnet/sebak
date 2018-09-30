package runner

import (
	"fmt"

	"boscoin.io/sebak/lib/ballot"
	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/error"
	"boscoin.io/sebak/lib/node"
	"boscoin.io/sebak/lib/storage"
	"boscoin.io/sebak/lib/transaction"
)

type BallotTransactionChecker struct {
	common.DefaultChecker

	NodeRunner *NodeRunner
	LocalNode  node.Node
	NetworkID  []byte

	Ballot               ballot.Ballot
	Transactions         []string
	VotingHole           ballot.VotingHole
	ValidTransactions    []string
	validTransactionsMap map[string]bool
	CheckAll             bool
}

func (checker *BallotTransactionChecker) InvalidTransactions() (invalids []string) {
	for _, hash := range checker.Transactions {
		if _, found := checker.validTransactionsMap[hash]; found {
			continue
		}

		invalids = append(invalids, hash)
	}

	return
}

func (checker *BallotTransactionChecker) setValidTransactions(hashes []string) {
	checker.ValidTransactions = hashes

	checker.validTransactionsMap = map[string]bool{}
	for _, hash := range hashes {
		checker.validTransactionsMap[hash] = true
	}

	return
}

// TransactionsIsNew checks the incoming transaction is
// already stored or not.
func IsNew(c common.Checker, args ...interface{}) (err error) {
	checker := c.(*BallotTransactionChecker)

	var validTransactions []string
	for _, hash := range checker.Transactions {
		// check transaction is already stored
		var found bool
		if found, err = block.ExistsBlockTransaction(checker.NodeRunner.Storage(), hash); err != nil || found {
			if !checker.CheckAll {
				err = errors.ErrorNewButKnownMessage
				return
			}
			continue
		}
		validTransactions = append(validTransactions, hash)
	}

	err = nil
	checker.setValidTransactions(validTransactions)

	return
}

// GetMissingTransaction will get the missing
// tranactions, that is, not in `TransactionPool` from proposer.
func GetMissingTransaction(c common.Checker, args ...interface{}) (err error) {
	checker := c.(*BallotTransactionChecker)

	var validTransactions []string
	for _, hash := range checker.ValidTransactions {
		if !checker.NodeRunner.Consensus().TransactionPool.Has(hash) {
			// TODO get transaction from proposer and check
			// `Transaction.IsWellFormed()`
			continue
		}
		validTransactions = append(validTransactions, hash)
	}

	checker.setValidTransactions(validTransactions)

	return
}

// BallotTransactionsSourceCheck checks there are transactions which has same
// source in the `Transactions`.
func BallotTransactionsSameSource(c common.Checker, args ...interface{}) (err error) {
	checker := c.(*BallotTransactionChecker)

	var validTransactions []string
	sources := map[string]bool{}
	for _, hash := range checker.ValidTransactions {
		tx, _ := checker.NodeRunner.Consensus().TransactionPool.Get(hash)
		if found := common.InStringMap(sources, tx.B.Source); found {
			if !checker.CheckAll {
				err = errors.ErrorTransactionSameSource
				return
			}
			continue
		}

		sources[tx.B.Source] = true
		validTransactions = append(validTransactions, hash)
	}
	err = nil
	checker.setValidTransactions(validTransactions)

	return
}

// BallotTransactionsSourceCheck calls `Transaction.Validate()`.
func BallotTransactionsSourceCheck(c common.Checker, args ...interface{}) (err error) {
	checker := c.(*BallotTransactionChecker)

	var validTransactions []string
	for _, hash := range checker.ValidTransactions {
		tx, _ := checker.NodeRunner.Consensus().TransactionPool.Get(hash)

		if err = ValidateTx(checker.NodeRunner.Storage(), tx); err != nil {
			if !checker.CheckAll {
				return
			}
			continue
		}
		validTransactions = append(validTransactions, hash)
	}

	err = nil
	checker.setValidTransactions(validTransactions)

	return
}

// BallotTransactionsOperationBodyCollectTxFee validates the
// `BallotTransactionsOperationBodyCollectTxFee.Amount` is matched with the
// collected fee of all transactions.
func BallotTransactionsOperationBodyCollectTxFee(c common.Checker, args ...interface{}) (err error) {
	checker := c.(*BallotTransactionChecker)

	var opb transaction.OperationBodyCollectTxFee
	if opb, err = checker.Ballot.ProposerTransaction().OperationBodyCollectTxFee(); err != nil {
		return
	}

	// check the colleted transaction fee is matched with
	// `OperationBodyCollectTxFee.Amount`
	if checker.Ballot.TransactionsLength() < 1 {
		if opb.Amount != 0 {
			err = errors.ErrorInvalidOperation
			return
		}
	} else {
		var fee common.Amount
		for _, hash := range checker.Transactions {
			if tx, found := checker.NodeRunner.Consensus().TransactionPool.Get(hash); !found {
				err = errors.ErrorTransactionNotFound
				return
			} else {
				fee = fee.MustAdd(tx.B.Fee)
			}
		}
		if opb.Amount != fee {
			err = errors.ErrorInvalidFee
			return
		}
	}

	return
}

//
// Validate the entirety of a transaction
//
// This function is critical for consensus, as it defines the validation rules for a transaction.
// It is executed whenever a transaction is received.
//
// As it is run on every single transaction, it should be as fast as possible,
// thus any new code should carefully consider when it takes place.
// Consider putting cheap, frequent errors first.
//
// Note that if we get to this function, it means the transaction is known to be
// well-formed (e.g. the signature is legit).
//
// Params:
//   st = Storage backend to use (e.g. to access the blocks)
//        Only ever read from, never written to.
//   tx = Transaction to check
//
func ValidateTx(st *storage.LevelDBBackend, tx transaction.Transaction) (err error) {
	// check, source exists
	var ba *block.BlockAccount
	if ba, err = block.GetBlockAccount(st, tx.B.Source); err != nil {
		err = errors.ErrorBlockAccountDoesNotExists
		return
	}

	// check, sequenceID is based on latest sequenceID
	if !tx.IsValidSequenceID(ba.SequenceID) {
		err = errors.ErrorTransactionInvalidSequenceID
		return
	}

	// get the balance at sequenceID
	var bac block.BlockAccountSequenceID
	bac, err = block.GetBlockAccountSequenceID(st, tx.B.Source, tx.B.SequenceID)
	if err != nil {
		return
	}

	totalAmount := tx.TotalAmount(true)

	// check, have enough balance at sequenceID
	if bac.Balance < totalAmount {
		fmt.Println("0000000", bac.Balance, totalAmount)
		err = errors.ErrorTransactionExcessAbilityToPay
		return
	}

	for _, op := range tx.B.Operations {
		if err = ValidateOp(st, ba, op); err != nil {
			return
		}
	}

	return
}

//
// Validate an operation
//
// This function is critical for consensus, as it defines the validation rules for an operation,
// and by extension a transaction. It is called from ValidateTx.
//
// Params:
//   st = Storage backend to use (e.g. to access the blocks)
//        Only ever read from, never written to.
//   source = Account from where the transaction (and ops) come from
//   tx = Transaction to check
//
func ValidateOp(st *storage.LevelDBBackend, source *block.BlockAccount, op transaction.Operation) (err error) {
	switch op.H.Type {
	case transaction.OperationCreateAccount:
		var ok bool
		var casted transaction.OperationBodyCreateAccount
		if casted, ok = op.B.(transaction.OperationBodyCreateAccount); !ok {
			err = errors.ErrorTypeOperationBodyNotMatched
			return
		}
		var exists bool
		if exists, err = block.ExistsBlockAccount(st, op.B.(transaction.OperationBodyCreateAccount).Target); err == nil && exists {
			err = errors.ErrorBlockAccountAlreadyExists
			return
		}
		// If it's a frozen account we check that only whole units are frozen
		if casted.Linked != "" && (casted.Amount%common.Unit) != 0 {
			return errors.ErrorFrozenAccountCreationWholeUnit // FIXME
		}
	case transaction.OperationPayment:
		var ok bool
		var casted transaction.OperationBodyPayment
		if casted, ok = op.B.(transaction.OperationBodyPayment); !ok {
			err = errors.ErrorTypeOperationBodyNotMatched
			return
		}
		var taccount *block.BlockAccount
		if taccount, err = block.GetBlockAccount(st, casted.Target); err != nil {
			err = errors.ErrorBlockAccountDoesNotExists
			return
		}
		// If it's a frozen account, it cannot receive payment
		if taccount.Linked != "" {
			err = errors.ErrorFrozenAccountNoDeposit
			return
		}
		// If it's a frozen account, everything must be withdrawn
		if source.Linked != "" {
			var expected common.Amount
			expected, err = source.Balance.Sub(common.BaseFee)
			if casted.Amount != expected {
				err = errors.ErrorFrozenAccountMustWithdrawEverything
				return
			}
		}
	case transaction.OperationCongressVoting, transaction.OperationCongressVotingResult:
		// Nothing to do
		return

	default:
		err = errors.ErrorUnknownOperationType
		return
	}
	return
}
