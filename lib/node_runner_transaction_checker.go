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

	Ballot            Ballot
	VotingHole        VotingHole
	ValidTransactions map[string]bool
	CheckAll          bool
}

func (checker *NodeRunnerHandleTransactionChecker) isValidTransaction(hash string) bool {
	return checker.Ballot.IsValidTransaction(hash)
}

func (checker *NodeRunnerHandleTransactionChecker) GetValidTransactionSlice() []string {
	validTransactionSlice := make([]string, 0, len(checker.ValidTransactions))
	for hash := range checker.ValidTransactions {
		validTransactionSlice = append(validTransactionSlice, hash)
	}
	return validTransactionSlice
}

func CheckNodeRunnerHandleTransactionsIsNew(c sebakcommon.Checker, args ...interface{}) (err error) {
	checker := c.(*NodeRunnerHandleTransactionChecker)

	validTransactions := make(map[string]bool)
	for _, hash := range checker.Ballot.Transactions() {
		// check transaction is already stored
		var found bool
		if found, err = ExistBlockTransaction(checker.NodeRunner.Storage(), hash); err != nil || found {
			if !checker.CheckAll && !checker.isValidTransaction(hash) {
				err = sebakerror.ErrorNewButKnownMessage
				return
			}
			continue
		}
		validTransactions[hash] = true
	}

	err = nil
	checker.ValidTransactions = validTransactions

	return
}

func CheckNodeRunnerHandleTransactionsGetMissingTransaction(c sebakcommon.Checker, args ...interface{}) (err error) {
	checker := c.(*NodeRunnerHandleTransactionChecker)

	validTransactions := make(map[string]bool)
	for hash := range checker.ValidTransactions {
		if !checker.NodeRunner.Consensus().TransactionPool.Has(hash) {
			// TODO get transaction from proposer and check
			// `Transaction.IsWellFormed()`
			continue
		}
		validTransactions[hash] = true
	}

	checker.ValidTransactions = validTransactions

	return
}

func CheckNodeRunnerHandleTransactionsSameSource(c sebakcommon.Checker, args ...interface{}) (err error) {
	checker := c.(*NodeRunnerHandleTransactionChecker)

	sources := map[string]bool{}
	validTransactions := make(map[string]bool)
	for hash := range checker.ValidTransactions {
		tx, _ := checker.NodeRunner.Consensus().TransactionPool.Get(hash)
		if found := sebakcommon.InStringMap(sources, tx.B.Source); found {
			if !checker.CheckAll && !checker.isValidTransaction(hash) {
				err = sebakerror.ErrorTransactionSameSource
				return
			}
			continue
		}

		sources[tx.B.Source] = true
		validTransactions[hash] = true
	}
	err = nil
	checker.ValidTransactions = validTransactions

	return
}

func checkTransactionSourceCheck(checker *NodeRunnerHandleTransactionChecker, tx Transaction) (err error) {
	// check, source exists
	var ba *BlockAccount
	if ba, err = GetBlockAccount(checker.NodeRunner.Storage(), tx.B.Source); err != nil {
		err = sebakerror.ErrorBlockAccountDoesNotExists
		return
	}

	// check, checkpoint is based on latest checkpoint
	if !tx.IsValidCheckpoint(ba.Checkpoint) {
		err = sebakerror.ErrorTransactionInvalidCheckpoint
		return
	}

	// get the balance at checkpoint
	var bac BlockAccountCheckpoint
	bac, err = GetBlockAccountCheckpoint(checker.NodeRunner.Storage(), tx.B.Source, tx.B.Checkpoint)
	if err != nil {
		return
	}

	totalAmount := tx.TotalAmount(true)

	// check, have enough balance at checkpoint
	if MustAmountFromString(bac.Balance) < totalAmount {
		err = sebakerror.ErrorTransactionExcessAbilityToPay
		return
	}

	// check, have enough balance now
	if MustAmountFromString(ba.Balance) < totalAmount {
		err = sebakerror.ErrorTransactionExcessAbilityToPay
		return
	}

	return
}

func CheckNodeRunnerHandleTransactionsSourceCheck(c sebakcommon.Checker, args ...interface{}) (err error) {
	checker := c.(*NodeRunnerHandleTransactionChecker)

	validTransactions := make(map[string]bool)
	for hash := range checker.ValidTransactions {
		tx, _ := checker.NodeRunner.Consensus().TransactionPool.Get(hash)

		if err = checkTransactionSourceCheck(checker, tx); err != nil {
			if !checker.CheckAll && !checker.isValidTransaction(hash) {
				return
			}
			continue
		}
		validTransactions[hash] = true
	}

	err = nil
	checker.ValidTransactions = validTransactions

	return
}
