package block

import (
	"encoding/json"
	"fmt"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/errors"
	"boscoin.io/sebak/lib/storage"
	"boscoin.io/sebak/lib/transaction"
	"boscoin.io/sebak/lib/transaction/operation"
)

// BlockOperation is `Operation` data for block. the storage should support,
//  * find by `Hash`
//  * find by `TxHash`
//
//  * get list by `Source` and created order
//  * get list by `Target` and created order

type BlockOperation struct {
	Hash string `json:"hash"`

	OpHash string `json:"op_hash"`
	TxHash string `json:"tx_hash"`

	Type   operation.OperationType `json:"type"`
	Source string                  `json:"source"`
	Body   []byte                  `json:"body"`
	Height uint64                  `json:"block_height"`

	// transaction will be used only for `Save` time.
	transaction transaction.Transaction
	isSaved     bool
}

func NewBlockOperationKey(opHash, txHash string) string {
	return fmt.Sprintf("%s-%s", opHash, txHash)
}

func NewBlockOperationFromOperation(op operation.Operation, tx transaction.Transaction, blockHeight uint64) (BlockOperation, error) {
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
		Height: blockHeight,

		transaction: tx,
	}, nil
}

func (bo *BlockOperation) Save(st *storage.LevelDBBackend) (err error) {
	if bo.isSaved {
		return errors.AlreadySaved
	}

	key := GetBlockOperationKey(bo.Hash)

	var exists bool
	if exists, err = st.Has(key); err != nil {
		return
	} else if exists {
		return errors.BlockAlreadyExists
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

	if err = st.New(bo.NewBlockOperationSourceAndTypeKey(), bo.Hash); err != nil {
		return
	}
	if err = st.New(bo.NewBlockOperationBlockHeightKey(), bo.Hash); err != nil {
		return
	}

	var body operation.Body
	var casted operation.CreateAccount
	var ok bool

	if bo.Type == operation.TypeCreateAccount {
		if body, err = operation.UnmarshalBodyJSON(bo.Type, bo.Body); err != nil {
			return err
		}
		if casted, ok = body.(operation.CreateAccount); !ok {
			return errors.TypeOperationBodyNotMatched
		}
		if casted.Linked != "" {
			if err = st.New(GetBlockOperationCreateFrozenKey(casted.Target, bo.Height), bo.Hash); err != nil {
				return err
			}
			if err = st.New(bo.NewBlockOperationFrozenLinkedKey(casted.Linked), bo.Hash); err != nil {
				return err
			}
		}
	}

	bo.isSaved = true

	return nil
}

func (bo BlockOperation) Serialize() (encoded []byte, err error) {
	encoded, err = common.EncodeJSONValue(bo)
	return
}

func GetBlockOperationKey(hash string) string {
	return fmt.Sprintf("%s%s", common.BlockOperationPrefixHash, hash)
}

func GetBlockOperationCreateFrozenKey(hash string, height uint64) string {
	return fmt.Sprintf(
		"%s%s%s",
		common.BlockOperationPrefixCreateFrozen,
		common.EncodeUint64ToByteSlice(height),
		hash,
	)
}

func GetBlockOperationKeyPrefixFrozenLinked(hash string) string {
	return fmt.Sprintf(
		"%s%s",
		common.BlockOperationPrefixFrozenLinked,
		hash,
	)
}

func GetBlockOperationKeyPrefixTxHash(txHash string) string {
	return fmt.Sprintf("%s%s-", common.BlockOperationPrefixTxHash, txHash)
}

func GetBlockOperationKeyPrefixSource(source string) string {
	return fmt.Sprintf("%s%s-", common.BlockOperationPrefixSource, source)
}

func GetBlockOperationKeyPrefixSourceAndType(source string, ty operation.OperationType) string {
	return fmt.Sprintf("%s%s%s-", common.BlockOperationPrefixSource, source, string(ty))
}

func GetBlockOperationKeyPrefixBlockHeight(height uint64) string {
	return fmt.Sprintf("%s%s-", common.BlockOperationPrefixBlockHeight, common.EncodeUint64ToByteSlice(height))
}

func (bo BlockOperation) NewBlockOperationTxHashKey() string {
	return fmt.Sprintf(
		"%s%s%s%s",
		GetBlockOperationKeyPrefixTxHash(bo.TxHash),
		common.EncodeUint64ToByteSlice(bo.Height),
		common.EncodeUint64ToByteSlice(bo.transaction.B.SequenceID),
		common.GetUniqueIDFromUUID(),
	)
}

func (bo BlockOperation) NewBlockOperationSourceKey() string {
	return fmt.Sprintf(
		"%s%s%s%s",
		GetBlockOperationKeyPrefixSource(bo.Source),
		common.EncodeUint64ToByteSlice(bo.Height),
		common.EncodeUint64ToByteSlice(bo.transaction.B.SequenceID),
		common.GetUniqueIDFromUUID(),
	)
}

func (bo BlockOperation) NewBlockOperationFrozenLinkedKey(hash string) string {
	return fmt.Sprintf(
		"%s%s",
		GetBlockOperationKeyPrefixFrozenLinked(hash),
		common.EncodeUint64ToByteSlice(bo.Height),
	)
}

func (bo BlockOperation) NewBlockOperationSourceAndTypeKey() string {
	return fmt.Sprintf(
		"%s%s%s%s",
		GetBlockOperationKeyPrefixSourceAndType(bo.Source, bo.Type),
		common.EncodeUint64ToByteSlice(bo.Height),
		common.EncodeUint64ToByteSlice(bo.transaction.B.SequenceID),
		common.GetUniqueIDFromUUID(),
	)
}

func (bo BlockOperation) NewBlockOperationBlockHeightKey() string {
	return fmt.Sprintf(
		"%s%s%s",
		GetBlockOperationKeyPrefixBlockHeight(bo.Height),
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

// Find all operations which created frozen account.
func GetBlockOperationsByFrozen(st *storage.LevelDBBackend, options storage.ListOptions) (
	func() (BlockOperation, bool, []byte),
	func(),
) {
	iterFunc, closeFunc := st.GetIterator(common.BlockOperationPrefixCreateFrozen, options)
	return LoadBlockOperationsInsideIterator(st, iterFunc, closeFunc)
}

// Find all operations which created frozen account and have the link of a general account's address.
func GetBlockOperationsByLinked(st *storage.LevelDBBackend, hash string, options storage.ListOptions) (
	func() (BlockOperation, bool, []byte),
	func(),
) {
	iterFunc, closeFunc := st.GetIterator(GetBlockOperationKeyPrefixFrozenLinked(hash), options)
	return LoadBlockOperationsInsideIterator(st, iterFunc, closeFunc)
}

func GetBlockOperationsBySourceAndType(st *storage.LevelDBBackend, source string, ty operation.OperationType, options storage.ListOptions) (
	func() (BlockOperation, bool, []byte),
	func(),
) {
	iterFunc, closeFunc := st.GetIterator(GetBlockOperationKeyPrefixSourceAndType(source, ty), options)
	return LoadBlockOperationsInsideIterator(st, iterFunc, closeFunc)
}

func GetBlockOperationsByBlockHeight(st *storage.LevelDBBackend, height uint64, options storage.ListOptions) (
	func() (BlockOperation, bool, []byte),
	func(),
) {
	iterFunc, closeFunc := st.GetIterator(GetBlockOperationKeyPrefixBlockHeight(height), options)
	return LoadBlockOperationsInsideIterator(st, iterFunc, closeFunc)
}
