package sebak

import (
	"encoding/json"
	"fmt"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/error"
	"boscoin.io/sebak/lib/observer"
	"boscoin.io/sebak/lib/storage"
)

// BlockTransaction is `Transaction` data for block. the storage should support,
//  * find by `Hash`
//
//  * get list by `Checkpoint` and created order
//  * get list by `Source` and created order
//  * get list by `Confirmed` order
//  * get list by `Account` and created order
//  * get list by `Block` and created order

const (
	BlockTransactionPrefixHash       string = "bt-hash-"       // bt-hash-<BlockTransaction.Hash>
	BlockTransactionPrefixCheckpoint string = "bt-checkpoint-" // bt-checkpoint-<BlockTransaction.SourceCheckpoint>,<BlockTransaction.TargetCheckpoint>
	BlockTransactionPrefixSource     string = "bt-source-"     // bt-source-<BlockTransaction.Source>
	BlockTransactionPrefixConfirmed  string = "bt-confirmed-"  // bt-confirmed-<BlockTransaction.Confirmed>
	BlockTransactionPrefixAccount    string = "bt-account-"    // bt-account-<BlockTransaction.Source>,<BlockTransaction.Operations.Target>
	BlockTransactionPrefixBlock      string = "bt-block-"      // bt-block-<BlockTransaction.Block>
)

// TODO(BlockTransaction): support counting

type BlockTransaction struct {
	Hash  string
	Block string /* `Block.Hash` */

	PreviousCheckpoint string
	SourceCheckpoint   string
	TargetCheckpoint   string
	Signature          string
	Source             string
	Fee                common.Amount
	Operations         []string
	Amount             common.Amount

	Confirmed string
	Created   string
	Message   []byte

	transaction Transaction
	isSaved     bool
}

func NewBlockTransactionFromTransaction(blockHash string, tx Transaction, message []byte) BlockTransaction {
	var opHashes []string
	for _, op := range tx.B.Operations {
		opHashes = append(opHashes, NewBlockOperationKey(op, tx))
	}

	return BlockTransaction{
		Hash:               tx.H.Hash,
		Block:              blockHash,
		PreviousCheckpoint: tx.B.Checkpoint,
		SourceCheckpoint:   tx.NextSourceCheckpoint(),
		TargetCheckpoint:   tx.NextTargetCheckpoint(),
		Signature:          tx.H.Signature,
		Source:             tx.B.Source,
		Fee:                tx.B.Fee,
		Operations:         opHashes,
		Amount:             tx.TotalAmount(true),

		Created: tx.H.Created,
		Message: message,

		transaction: tx,
	}
}

func GetBlockTransactionKeyCheckpoint(checkpoint string) string {
	return fmt.Sprintf("%s%s", BlockTransactionPrefixCheckpoint, checkpoint)
}

func (bt BlockTransaction) NewBlockTransactionKeySource() string {
	return fmt.Sprintf(
		"%s%s",
		GetBlockTransactionKeyPrefixSource(bt.Source),
		common.GetUniqueIDFromUUID(),
	)
}

func (bt BlockTransaction) NewBlockTransactionKeyConfirmed() string {
	return fmt.Sprintf(
		"%s%s",
		GetBlockTransactionKeyPrefixConfirmed(bt.Confirmed),
		common.GetUniqueIDFromUUID(),
	)
}

func (bt BlockTransaction) NewBlockTransactionKeyByAccount(accountAddress string) string {
	return fmt.Sprintf(
		"%s%s",
		GetBlockTransactionKeyPrefixAccount(accountAddress),
		common.GetUniqueIDFromUUID(),
	)
}

func (bt BlockTransaction) NewBlockTransactionKeyByBlock(hash string) string {
	return fmt.Sprintf(
		"%s%s",
		GetBlockTransactionKeyPrefixBlock(hash),
		common.GetUniqueIDFromUUID(),
	)
}

func (bt *BlockTransaction) Save(st *sebakstorage.LevelDBBackend) (err error) {
	if bt.isSaved {
		return errors.ErrorAlreadySaved
	}

	key := GetBlockTransactionKey(bt.Hash)

	var exists bool
	exists, err = st.Has(key)
	if err != nil {
		return
	} else if exists {
		return errors.ErrorBlockAlreadyExists
	}

	bt.Confirmed = common.NowISO8601()
	if err = st.New(GetBlockTransactionKey(bt.Hash), bt); err != nil {
		return
	}
	if err = st.New(GetBlockTransactionKeyCheckpoint(bt.SourceCheckpoint), bt.Hash); err != nil {
		return
	}
	if err = st.New(GetBlockTransactionKeyCheckpoint(bt.TargetCheckpoint), bt.Hash); err != nil {
		return
	}
	if err = st.New(bt.NewBlockTransactionKeySource(), bt.Hash); err != nil {
		return
	}
	if err = st.New(bt.NewBlockTransactionKeyConfirmed(), bt.Hash); err != nil {
		return
	}
	if err = st.New(bt.NewBlockTransactionKeyByAccount(bt.Source), bt.Hash); err != nil {
		return
	}
	if err = st.New(bt.NewBlockTransactionKeyByBlock(bt.Block), bt.Hash); err != nil {
		return
	}
	for _, op := range bt.transaction.B.Operations {
		bo := NewBlockOperationFromOperation(op, bt.transaction)
		if err = bo.Save(st); err != nil {
			return
		}

		target := op.B.TargetAddress()
		if err = st.New(bt.NewBlockTransactionKeyByAccount(target), bt.Hash); err != nil {
			return
		}
	}
	event := "saved"
	event += " " + fmt.Sprintf("source-%s", bt.Source)
	event += " " + fmt.Sprintf("hash-%s", bt.Hash)
	observer.BlockTransactionObserver.Trigger(event, bt)
	bt.isSaved = true

	return nil
}

func (bt BlockTransaction) Serialize() (encoded []byte, err error) {
	encoded, err = common.EncodeJSONValue(bt)
	return
}

func (bt BlockTransaction) String() string {
	encoded, _ := common.EncodeJSONValue(bt)
	return string(encoded)
}

func (bt BlockTransaction) Transaction() Transaction {
	return bt.transaction
}

func GetBlockTransactionKeyPrefixSource(source string) string {
	return fmt.Sprintf("%s%s-", BlockTransactionPrefixSource, source)
}

func GetBlockTransactionKeyPrefixConfirmed(confirmed string) string {
	return fmt.Sprintf("%s%s-", BlockTransactionPrefixConfirmed, confirmed)
}

func GetBlockTransactionKeyPrefixAccount(accountAddress string) string {
	return fmt.Sprintf("%s%s-", BlockTransactionPrefixAccount, accountAddress)
}

func GetBlockTransactionKeyPrefixBlock(hash string) string {
	return fmt.Sprintf("%s%s-", BlockTransactionPrefixBlock, hash)
}

func GetBlockTransactionKey(hash string) string {
	return fmt.Sprintf("%s%s", BlockTransactionPrefixHash, hash)
}

func GetBlockTransaction(st *sebakstorage.LevelDBBackend, hash string) (bt BlockTransaction, err error) {
	if err = st.Get(GetBlockTransactionKey(hash), &bt); err != nil {
		return
	}

	bt.isSaved = true
	return
}

func ExistBlockTransaction(st *sebakstorage.LevelDBBackend, hash string) (bool, error) {
	return st.Has(GetBlockTransactionKey(hash))
}

func LoadBlockTransactionsInsideIterator(
	st *sebakstorage.LevelDBBackend,
	iterFunc func() (sebakstorage.IterItem, bool),
	closeFunc func(),
) (
	func() (BlockTransaction, bool),
	func(),
) {

	return (func() (BlockTransaction, bool) {
			item, hasNext := iterFunc()
			if !hasNext {
				return BlockTransaction{}, false
			}

			var hash string
			json.Unmarshal(item.Value, &hash)

			bt, err := GetBlockTransaction(st, hash)
			if err != nil {
				return BlockTransaction{}, false
			}

			return bt, hasNext
		}), (func() {
			closeFunc()
		})
}

func GetBlockTransactionByCheckpoint(st *sebakstorage.LevelDBBackend, checkpoint string) (bt BlockTransaction, err error) {
	var hash string
	if err = st.Get(GetBlockTransactionKeyCheckpoint(checkpoint), &hash); err != nil {
		return
	}
	bt, err = GetBlockTransaction(st, hash)

	return
}

func GetBlockTransactionsBySource(st *sebakstorage.LevelDBBackend, source string, reverse bool) (
	func() (BlockTransaction, bool),
	func(),
) {
	iterFunc, closeFunc := st.GetIterator(GetBlockTransactionKeyPrefixSource(source), reverse)

	return LoadBlockTransactionsInsideIterator(st, iterFunc, closeFunc)
}

func GetBlockTransactionsByConfirmed(st *sebakstorage.LevelDBBackend, reverse bool) (
	func() (BlockTransaction, bool),
	func(),
) {
	iterFunc, closeFunc := st.GetIterator(BlockTransactionPrefixConfirmed, reverse)

	return LoadBlockTransactionsInsideIterator(st, iterFunc, closeFunc)
}

func GetBlockTransactionsByAccount(st *sebakstorage.LevelDBBackend, accountAddress string, reverse bool) (
	func() (BlockTransaction, bool),
	func(),
) {
	iterFunc, closeFunc := st.GetIterator(GetBlockTransactionKeyPrefixAccount(accountAddress), reverse)
	return LoadBlockTransactionsInsideIterator(st, iterFunc, closeFunc)
}

func GetBlockTransactionsByBlock(st *sebakstorage.LevelDBBackend, hash string, reverse bool) (
	func() (BlockTransaction, bool),
	func(),
) {
	iterFunc, closeFunc := st.GetIterator(GetBlockTransactionKeyPrefixBlock(hash), reverse)
	return LoadBlockTransactionsInsideIterator(st, iterFunc, closeFunc)
}

var GetBlockTransactions = GetBlockTransactionsByConfirmed
