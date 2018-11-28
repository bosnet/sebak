/*
	This file contains features that, define the actions that a node should do when it receives a message.
	The MessageChecker struct is a data structure that is shared while the checker methods are running.
	Checker methods are called sequentially by the RunChecker() method in the handleMessageFromClient method of noderunner.go.
	The process is as follows :
	1. TransactionUnmarshal: Unmarshal the received message in transaction
	2. HasTransaction: The transaction that already exists does not proceed anymore
	3. SaveTransactionHistory: Save History
	4. PushIntoTransactionPool: Insert into transaction pool
	5. BroadcastTransaction: Passing a transaction to all known Validators.
*/

package runner

import (
	"math/rand"

	logging "github.com/inconshreveable/log15"

	"encoding/json"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/consensus"
	"boscoin.io/sebak/lib/errors"
	"boscoin.io/sebak/lib/node"
	"boscoin.io/sebak/lib/storage"
	"boscoin.io/sebak/lib/transaction"
)

type MessageChecker struct {
	common.DefaultChecker

	Conf            common.Config
	LocalNode       *node.LocalNode
	NetworkID       []byte
	Message         common.NetworkMessage
	Log             logging.Logger
	Consensus       *consensus.ISAAC
	TransactionPool *transaction.Pool
	Storage         *storage.LevelDBBackend
	Transaction     transaction.Transaction
}

// TransactionUnmarshal makes `Transaction` from
// incoming `common.NetworkMessage`.
func TransactionUnmarshal(c common.Checker, args ...interface{}) (err error) {
	checker := c.(*MessageChecker)

	var tx transaction.Transaction
	if err = json.Unmarshal(checker.Message.Data, &tx); err != nil {
		return
	}

	if err = tx.IsWellFormed(checker.Conf); err != nil {
		return
	}

	checker.Transaction = tx
	checker.Log = checker.Log.New(logging.Ctx{"transaction": tx.GetHash()})
	checker.Log.Debug("message is transaction")

	return
}

// HasTransaction checks transaction is in
// `Pool` And `Block`
func HasTransaction(c common.Checker, args ...interface{}) (err error) {
	checker := c.(*MessageChecker)

	hash := checker.Transaction.GetHash()

	if checker.TransactionPool.Has(hash) {
		return errors.NewButKnownMessage
	}

	if exists, err := block.ExistsBlockTransaction(checker.Storage, hash); err != nil {
		return err
	} else if exists {
		return errors.NewButKnownMessage
	}

	return nil
}

// SameSource checks there are transactions which has same source in the
// `Pool`.
func MessageHasSameSource(c common.Checker, args ...interface{}) (err error) {
	checker := c.(*MessageChecker)

	if checker.TransactionPool.IsSameSource(checker.Transaction.Source()) {
		err = errors.TransactionSameSourceInPool
		return
	}

	return
}

// MessageValidate validates.
func MessageValidate(c common.Checker, args ...interface{}) (err error) {
	checker := c.(*MessageChecker)

	if err = ValidateTx(checker.Storage, checker.Conf, checker.Transaction); err != nil {
		return
	}

	return
}

// PushIntoTransactionPool add the incoming
// transactions into `Pool`.
func PushIntoTransactionPool(c common.Checker, args ...interface{}) error {
	checker := c.(*MessageChecker)

	tx := checker.Transaction
	err := checker.TransactionPool.Add(tx)
	if err == errors.TransactionPoolFull {
		return err
	}

	if _, err = block.SaveTransactionPool(checker.Storage, tx); err != nil {
		return err
	}

	checker.Log.Debug("push transaction into TransactionPool")

	return nil
}

// PushIntoTransactionPoolFromClient add the incoming tx from client
func PushIntoTransactionPoolFromClient(c common.Checker, args ...interface{}) error {
	checker := c.(*MessageChecker)

	tx := checker.Transaction
	err := checker.TransactionPool.AddFromClient(tx)
	if err == errors.TransactionPoolFull {
		return err
	}

	if _, err = block.SaveTransactionPool(checker.Storage, tx); err != nil {
		return err
	}

	checker.Log.Debug("push transaction into TransactionPool from client")

	return nil
}

// PushIntoTransactionPoolFromNode add the incoming tx from node
func PushIntoTransactionPoolFromNode(c common.Checker, args ...interface{}) error {
	checker := c.(*MessageChecker)

	tx := checker.Transaction
	err := checker.TransactionPool.AddFromNode(tx)
	if err == errors.TransactionPoolFull {
		return err
	}

	if _, err = block.SaveTransactionPool(checker.Storage, tx); err != nil {
		return err
	}

	checker.Log.Debug("push transaction into TransactionPool from node")

	return nil
}

// BroadcastTransaction broadcasts the incoming
// transaction to the other nodes.
func BroadcastTransaction(c common.Checker, args ...interface{}) (err error) {
	checker := c.(*MessageChecker)

	checker.Log.Debug("transaction from client will be broadcasted")

	checker.Consensus.ConnectionManager().Broadcast(checker.Transaction)

	return
}

// BroadcastTransactionFromWatcher is sending tx to one of validators.
// If all validators returns error, it returns error.
func BroadcastTransactionFromWatcher(c common.Checker, args ...interface{}) error {
	checker := c.(*MessageChecker)
	if checker.Conf.WatcherMode == false {
		return nil
	}
	checker.Log.Debug("transaction from client will be sent")

	cm := checker.Consensus.ConnectionManager()
	var addrs []string

	for _, a := range cm.AllConnected() {
		if a != checker.LocalNode.Address() {
			addrs = append(addrs, a)
		}
	}

	if len(addrs) <= 0 {
		return errors.AllValidatorsNotConnected
	}

	raddrs := make([]string, len(addrs))
	perm := rand.Perm(len(addrs))
	for i, v := range perm {
		raddrs[v] = addrs[i]
	}

	var err error
	for _, a := range raddrs {
		client := cm.GetConnection(a)
		_, err = client.SendTransaction(checker.Transaction)
		if err == nil {
			// Broaadcast from watcher is that send one of them using client api successfully.
			checker.Log.Info("send tx to node", "node", a, "tx", checker.Transaction.GetHash())
			break
		}
		checker.Log.Debug("failure to send tx to node", "node", a, "err", err, "tx", checker.Transaction.GetHash())
	}
	return err
}
