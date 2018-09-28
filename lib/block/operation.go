package block

import (
	"encoding/json"
	"fmt"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/common/observer"
	"boscoin.io/sebak/lib/error"
	"boscoin.io/sebak/lib/storage"
	"boscoin.io/sebak/lib/transaction"
)

// BlockOperation is `Operation` data for block. the storage should support,
//  * find by `Hash`
//  * find by `TxHash`
//
//  * get list by `Source` and created order
//  * get list by `Target` and created order

type BlockOperation struct {
	Hash string

	OpHash string
	TxHash string

	Type   transaction.OperationType
	Source string
	Body   []byte

	// transaction will be used only for `Save` time.
	transaction transaction.Transaction
	isSaved     bool
	blockHeight uint64
}

func NewBlockOperationKey(opHash, txHash string) string {
	return fmt.Sprintf("%s-%s", opHash, txHash)
}

func NewBlockOperationFromOperation(op transaction.Operation, tx transaction.Transaction, blockHeight uint64) (BlockOperation, error) {
	body, err := op.B.Serialize()
	if err != nil {
		return BlockOperation{}, err
	}

	opHash := op.MakeHashString()
	txHash := tx.GetHash()

	return BlockOperation{
		Hash: NewBlockOperationKey(opHash, txHash),

		OpHash: opHash,
		TxHash: txHash,

		Type:   op.H.Type,
		Source: tx.B.Source,
		Body:   body,

		transaction: tx,

		blockHeight: blockHeight,
	}, nil
}

func (bo *BlockOperation) Save(st *storage.LevelDBBackend) (err error) {
	if bo.isSaved {
		return errors.ErrorAlreadySaved
	}

	key := GetBlockOperationKey(bo.Hash)

	var exists bool
	if exists, err = st.Has(key); err != nil {
		return
	} else if exists {
		return errors.ErrorBlockAlreadyExists
	}

	if err = st.New(key, bo); err != nil {
		return
	}
	if err = st.New(bo.NewBlockOperationTxHashKey(), bo.Hash); err != nil {
		return
	}
	if err = st.New(bo.NewBlockOperationSourceKey(), bo.Hash); err != nil {
		return
	}
	bo.isSaved = true

	event := "saved"
	event += " " + fmt.Sprintf("source-%s", bo.Source)
	event += " " + fmt.Sprintf("hash-%s", bo.Hash)
	event += " " + fmt.Sprintf("txhash-%s", bo.TxHash)
	observer.BlockOperationObserver.Trigger(event, bo)

	return nil
}

func (bo BlockOperation) Serialize() (encoded []byte, err error) {
	encoded, err = common.EncodeJSONValue(bo)
	return
}

func (bo BlockOperation) Transaction() transaction.Transaction {
	return bo.transaction
}

func GetBlockOperationKey(hash string) string {
	return fmt.Sprintf("%s%s", common.BlockOperationPrefixHash, hash)
}

func GetBlockOperationKeyPrefixTxHash(txHash string) string {
	return fmt.Sprintf("%s%s-", common.BlockOperationPrefixTxHash, txHash)
}

func GetBlockOperationKeyPrefixSource(source string) string {
	return fmt.Sprintf("%s%s-", common.BlockOperationPrefixSource, source)
}

func (bo BlockOperation) NewBlockOperationTxHashKey() string {
	return fmt.Sprintf(
		"%s%s%s%s",
		GetBlockOperationKeyPrefixTxHash(bo.TxHash),
		common.EncodeUint64ToByteSlice(bo.blockHeight),
		common.EncodeUint64ToByteSlice(bo.transaction.B.SequenceID),
		common.GetUniqueIDFromUUID(),
	)
}

func (bo BlockOperation) NewBlockOperationSourceKey() string {
	return fmt.Sprintf(
		"%s%s%s%s",
		GetBlockOperationKeyPrefixSource(bo.Source),
		common.EncodeUint64ToByteSlice(bo.blockHeight),
		common.EncodeUint64ToByteSlice(bo.transaction.B.SequenceID),
		common.GetUniqueIDFromUUID(),
	)
}

func ExistsBlockOperation(st *storage.LevelDBBackend, hash string) (bool, error) {
	return st.Has(GetBlockOperationKey(hash))
}

func GetBlockOperation(st *storage.LevelDBBackend, hash string) (bo BlockOperation, err error) {
	if err = st.Get(GetBlockOperationKey(hash), &bo); err != nil {
		return
	}

	bo.isSaved = true
	return
}

func LoadBlockOperationsInsideIterator(
	st *storage.LevelDBBackend,
	iterFunc func() (storage.IterItem, bool),
	closeFunc func(),
) (
	func() (BlockOperation, bool, []byte),
	func(),
) {

	return (func() (BlockOperation, bool, []byte) {
			item, hasNext := iterFunc()
			if !hasNext {
				return BlockOperation{}, false, item.Key
			}

			var hash string
			json.Unmarshal(item.Value, &hash)

			bo, err := GetBlockOperation(st, hash)
			if err != nil {
				return BlockOperation{}, false, item.Key
			}

			return bo, hasNext, item.Key
		}), (func() {
			closeFunc()
		})
}

func GetBlockOperationsByTxHash(st *storage.LevelDBBackend, txHash string, options storage.ListOptions) (
	func() (BlockOperation, bool, []byte),
	func(),
) {
	iterFunc, closeFunc := st.GetIterator(GetBlockOperationKeyPrefixTxHash(txHash), options)

	return LoadBlockOperationsInsideIterator(st, iterFunc, closeFunc)
}

func GetBlockOperationsBySource(st *storage.LevelDBBackend, source string, options storage.ListOptions) (
	func() (BlockOperation, bool, []byte),
	func(),
) {
	iterFunc, closeFunc := st.GetIterator(GetBlockOperationKeyPrefixSource(source), options)

	return LoadBlockOperationsInsideIterator(st, iterFunc, closeFunc)
}
