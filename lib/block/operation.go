package block

import (
	"encoding/json"
	"fmt"
	"sort"

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
	Hash   string
	TxHash string

	Type   transaction.OperationType
	Source string
	Target string
	Amount common.Amount

	// transaction will be used only for `Save` time.
	transaction transaction.Transaction
	isSaved     bool
	blockHeight uint64
}

func NewBlockOperationKey(op transaction.Operation, tx transaction.Transaction) string {
	return fmt.Sprintf("%s-%s", op.MakeHashString(), tx.GetHash())
}

func NewBlockOperationFromOperation(op transaction.Operation, tx transaction.Transaction, blockHeight uint64) BlockOperation {
	return BlockOperation{
		Hash:   NewBlockOperationKey(op, tx),
		TxHash: tx.H.Hash,

		Type:   op.H.Type,
		Source: tx.B.Source,
		Target: op.B.TargetAddress(),
		Amount: op.B.GetAmount(),

		transaction: tx,

		blockHeight: blockHeight,
	}
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

	if err = st.New(GetBlockOperationKey(bo.Hash), bo); err != nil {
		return
	}
	if err = st.New(bo.NewBlockOperationTxHashKey(), bo.Hash); err != nil {
		return
	}
	if err = st.New(bo.NewBlockOperationSourceKey(), bo.Hash); err != nil {
		return
	}
	if err = st.New(bo.NewBlockOperationTargetKey(), bo.Hash); err != nil {
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

func GetBlockOperationKeyPrefixTarget(target string) string {
	return fmt.Sprintf("%s%s-", common.BlockOperationPrefixTarget, target)
}

func GetBlockOperationKeyPrefixPeers(one, two string) string {
	return fmt.Sprintf("%s%s%s-", common.BlockOperationPrefixPeers, one, two)
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

func (bo BlockOperation) NewBlockOperationTargetKey() string {
	return fmt.Sprintf(
		"%s%s%s%s",
		GetBlockOperationKeyPrefixTarget(bo.Target),
		common.EncodeUint64ToByteSlice(bo.blockHeight),
		common.EncodeUint64ToByteSlice(bo.transaction.B.SequenceID),
		common.GetUniqueIDFromUUID(),
	)
}

func (bo BlockOperation) NewBlockOperationPeersKey() string {
	addresses := []string{bo.Target, bo.Source}
	sort.Strings(addresses)
	return fmt.Sprintf(
		"%s%s%s%s",
		GetBlockOperationKeyPrefixPeers(addresses[0], addresses[1]),
		common.EncodeUint64ToByteSlice(bo.blockHeight),
		common.EncodeUint64ToByteSlice(bo.transaction.B.SequenceID),
		common.GetUniqueIDFromUUID(),
	)
}

func ExistBlockOperation(st *storage.LevelDBBackend, hash string) (bool, error) {
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

func GetBlockOperationsByTarget(st *storage.LevelDBBackend, target string, options storage.ListOptions) (
	func() (BlockOperation, bool, []byte),
	func(),
) {
	iterFunc, closeFunc := st.GetIterator(GetBlockOperationKeyPrefixTarget(target), options)

	return LoadBlockOperationsInsideIterator(st, iterFunc, closeFunc)
}
