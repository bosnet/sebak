package sebak

import (
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/error"
	"boscoin.io/sebak/lib/node"
)

type BallotTransactionChecker struct {
	sebakcommon.DefaultChecker

	NodeRunner *NodeRunner
	LocalNode  sebaknode.Node
	NetworkID  []byte

	Transactions         []string
	VotingHole           sebakcommon.VotingHole
	ValidTransactions    []string
	validTransactionsMap map[string]bool
	CheckAll             bool
}

// TransactionsIsNew checks the incoming transaction is
// already stored or not.
func IsNew(c sebakcommon.Checker, args ...interface{}) (err error) {
	checker := c.(*BallotTransactionChecker)

	var validTransactions []string
	for _, hash := range checker.Transactions {
		// check transaction is already stored
		var found bool
		if found, err = ExistBlockTransaction(checker.NodeRunner.Storage(), hash); err != nil || found {
			if !checker.CheckAll {
				err = sebakerror.ErrorNewButKnownMessage
				return
			}
			continue
		}
		validTransactions = append(validTransactions, hash)
	}

	err = nil
	checker.ValidTransactions = validTransactions

	return
}

// GetMissingTransaction will get the missing
// tranactions, that is, not in `TransactionPool` from proposer.
func GetMissingTransaction(c sebakcommon.Checker, args ...interface{}) (err error) {
	checker := c.(*BallotTransactionChecker)

	var validTransactions []string
	//sources := map[string]bool{}
	for _, hash := range checker.ValidTransactions {
		var checkMissingTransaction = checker.NodeRunner.Consensus().TransactionPool.Has(hash)
		if !checkMissingTransaction {
			// TODO get transaction from proposer and check
			// `Transaction.IsWellFormed()`

			// continue
		}
		validTransactions = append(validTransactions, hash)
	}

	checker.ValidTransactions = validTransactions

	return
}

// TransactionsSameSource checks there are transactions
// which has same source in the `Transactions`.
func SameSource(c sebakcommon.Checker, args ...interface{}) (err error) {
	checker := c.(*BallotTransactionChecker)

	var validTransactions []string
	sources := map[string]bool{}
	for _, hash := range checker.ValidTransactions {
		tx, _ := checker.NodeRunner.Consensus().TransactionPool.Get(hash)
		if found := sebakcommon.InStringMap(sources, tx.B.Source); found {
			if !checker.CheckAll {
				err = sebakerror.ErrorTransactionSameSource
				return
			}
			continue
		}

		sources[tx.B.Source] = true
		validTransactions = append(validTransactions, hash)
	}
	err = nil
	checker.ValidTransactions = validTransactions

	return
}

// SourceCheck calls `Transaction.Validate()`.
func SourceCheck(c sebakcommon.Checker, args ...interface{}) (err error) {
	checker := c.(*BallotTransactionChecker)

	var validTransactions []string
	for _, hash := range checker.ValidTransactions {
		tx, _ := checker.NodeRunner.Consensus().TransactionPool.Get(hash)

		if err = tx.Validate(checker.NodeRunner.Storage()); err != nil {
			if !checker.CheckAll {
				return
			}
			continue
		}
		validTransactions = append(validTransactions, hash)
	}

	err = nil
	checker.ValidTransactions = validTransactions

	return
}
