/*
	This file contains features that, define the actions that a node should do when it receives a message.
	The MessageChecker struct is a data structure that is shared while the checker methods are running.
	Checker methods are called sequentially by the RunChecker() method in the handleMessageFromClient method of node_runner.go.
	The process is as follows :
	1. TransactionUnmarshal: Unmarshal the received message in transaction
	2. HasTransaction: The transaction that already exists does not proceed anymore
	3. SaveTransactionHistory: Save History
	4. PushIntoTransactionPool: Insert into transaction pool
	5. BroadcastTransaction: Passing a transaction to all known Validators.
*/

package sebak

import (
	logging "github.com/inconshreveable/log15"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/error"
	"boscoin.io/sebak/lib/node"
	"boscoin.io/sebak/lib/transaction"
)

type MessageChecker struct {
	common.DefaultChecker

	NodeRunner *NodeRunner
	LocalNode  *node.LocalNode
	NetworkID  []byte
	Message    common.NetworkMessage

	Log logging.Logger
	Transaction transaction.Transaction
}

// TransactionUnmarshal makes `Transaction` from
// incoming `common.NetworkMessage`.
func TransactionUnmarshal(c common.Checker, args ...interface{}) (err error) {
	checker := c.(*MessageChecker)

	var tx transaction.Transaction
	if tx, err = transaction.NewTransactionFromJSON(checker.Message.Data); err != nil {
		return
	}

	if err = tx.IsWellFormed(checker.NetworkID); err != nil {
		return
	}

	checker.Transaction = tx
	checker.Log = checker.NodeRunner.Log().New(logging.Ctx{"transaction": tx.GetHash()})
	checker.Log.Debug("message is transaction")

	return
}

// HasTransaction checks transaction is in
// `TransactionPool`.
func HasTransaction(c common.Checker, args ...interface{}) (err error) {
	checker := c.(*MessageChecker)

	consensus := checker.NodeRunner.Consensus()
	if consensus.TransactionPool.Has(checker.Transaction.GetHash()) {
		err = errors.ErrorNewButKnownMessage
		return
	}

	return
}

// SaveTransactionHistory checks transaction is in
// `BlockTransactionHistory`, which has the received transaction recently.
func SaveTransactionHistory(c common.Checker, args ...interface{}) (err error) {
	checker := c.(*MessageChecker)

	var found bool
	if found, err = ExistsBlockTransactionHistory(checker.NodeRunner.Storage(), checker.Transaction.GetHash()); found && err == nil {
		checker.Log.Debug("found in history")
		err = errors.ErrorNewButKnownMessage
		return
	}

	bt := NewTransactionHistoryFromTransaction(checker.Transaction, checker.Message.Data)
	if err = bt.Save(checker.NodeRunner.Storage()); err != nil {
		return
	}

	checker.Log.Debug("saved in history")

	return
}

// SameSource checks there are transactions which has same source in the
// `TransactionPool`.
func MessageHasSameSource(c common.Checker, args ...interface{}) (err error) {
	checker := c.(*MessageChecker)

	if checker.NodeRunner.Consensus().TransactionPool.IsSameSource(checker.Transaction.Source()) {
		err = errors.ErrorTransactionSameSource
		return
	}

	return
}

// MessageValidate validates.
func MessageValidate(c common.Checker, args ...interface{}) (err error) {
	checker := c.(*MessageChecker)

	if err = checker.Transaction.Validate(checker.NodeRunner.Storage()); err != nil {
		return
	}

	return
}

// PushIntoTransactionPool add the incoming
// transactions into `TransactionPool`.
func PushIntoTransactionPool(c common.Checker, args ...interface{}) (err error) {
	checker := c.(*MessageChecker)

	tx := checker.Transaction
	is := checker.NodeRunner.Consensus()
	is.TransactionPool.Add(tx)

	checker.Log.Debug("push transaction into transactionPool")

	return
}

// BroadcastTransaction broadcasts the incoming
// transaction to the other nodes.
func BroadcastTransaction(c common.Checker, args ...interface{}) (err error) {
	checker := c.(*MessageChecker)

	checker.Log.Debug("transaction from client will be broadcasted")

	// TODO sender should be excluded
	checker.NodeRunner.ConnectionManager().Broadcast(checker.Transaction)

	return
}
