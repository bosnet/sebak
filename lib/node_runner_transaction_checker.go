package sebak

import (
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/error"
	"boscoin.io/sebak/lib/node"
)

type NodeRunnerHandleTransactionChecker struct {
	sebakcommon.DefaultChecker

	NodeRunner *NodeRunner
	LocalNode  sebaknode.Node
	NetworkID  []byte

	Transactions         []string
	VotingHole           VotingHole
	ValidTransactions    []string
	validTransactionsMap map[string]bool
	CheckAll             bool
}

// CheckNodeRunnerHandleTransactionsIsNew checks the incoming transaction is
// already stored or not.
func CheckNodeRunnerHandleTransactionsIsNew(c sebakcommon.Checker, args ...interface{}) (err error) {
	checker := c.(*NodeRunnerHandleTransactionChecker)

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

// CheckNodeRunnerHandleTransactionsGetMissingTransaction will get the missing
// tranactions, that is, not in `TransactionPool` from proposer.
func CheckNodeRunnerHandleTransactionsGetMissingTransaction(c sebakcommon.Checker, args ...interface{}) (err error) {
	checker := c.(*NodeRunnerHandleTransactionChecker)

	var validTransactions []string
	for _, hash := range checker.ValidTransactions {
		if !checker.NodeRunner.Consensus().TransactionPool.Has(hash) {
			// TODO get transaction from proposer and check
			// `Transaction.IsWellFormed()`

			// continue
		}
		validTransactions = append(validTransactions, hash)
	}

	checker.ValidTransactions = validTransactions

	return
}

// CheckNodeRunnerHandleTransactionsSameSource checks there are transactions
// which has same source in the `Transactions`.
func CheckNodeRunnerHandleTransactionsSameSource(c sebakcommon.Checker, args ...interface{}) (err error) {
	checker := c.(*NodeRunnerHandleTransactionChecker)

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

// CheckNodeRunnerHandleTransactionsSourceCheck calls `Transaction.Validate()`.
func CheckNodeRunnerHandleTransactionsSourceCheck(c sebakcommon.Checker, args ...interface{}) (err error) {
	checker := c.(*NodeRunnerHandleTransactionChecker)

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
