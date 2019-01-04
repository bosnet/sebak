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
	Target string                  `json:"target"`
	Body   []byte                  `json:"body"`
	Height uint64                  `json:"block_height"`

	// bellows will be used only for `Save` time.
	transaction transaction.Transaction
	operation   operation.Operation
	linked      string
	isSaved     bool
	opIndex     int
}

func NewBlockOperationKey(opHash, txHash string) string {
	return fmt.Sprintf("%s-%s", opHash, txHash)
}

func NewBlockOperationFromOperation(op operation.Operation, tx transaction.Transaction, blockHeight uint64, opIndex int) (BlockOperation, error) {
	body, err := json.Marshal(op.B)
	if err != nil {
		return BlockOperation{}, err
	}

	opHash := common.MustMakeObjectHashString(op)
	txHash := tx.GetHash()

	target := ""
	if pop, ok := op.B.(operation.Targetable); ok {
		target = pop.TargetAddress()
	}

	linked := ""
	if createAccount, ok := op.B.(operation.CreateAccount); ok {
		if createAccount.Linked != "" {
			linked = createAccount.Linked
		}
	}

	return BlockOperation{
		Hash: NewBlockOperationKey(opHash, txHash),

		OpHash: opHash,
		TxHash: txHash,

		Type:   op.H.Type,
		Source: tx.B.Source,
		Target: target,
		Body:   body,
		Height: blockHeight,

		transaction: tx,
		operation:   op,
		linked:      linked,
		opIndex:     opIndex,
	}, nil
}

func (bo *BlockOperation) hasTarget() bool {
	if bo.Target != "" {
		return true
	}
	return false
}

func (bo *BlockOperation) targetIsLinked() bool {
	if bo.hasTarget() && bo.linked != "" {
		return true
	}
	return false
}

func (bo *BlockOperation) Save(st *storage.LevelDBBackend) (err error) {
	if bo.isSaved {
		return errors.AlreadySaved
	}

	key := key(bo.Hash)

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
	if err = st.New(bo.NewBlockOperationPeersKey(bo.Source), bo.Hash); err != nil {
		return
	}
	if err = st.New(bo.NewBlockOperationPeersAndTypeKey(bo.Source), bo.Hash); err != nil {
		return
	}
	if err = st.New(bo.NewBlockOperationBlockHeightKey(), bo.Hash); err != nil {
		return
	}

	if bo.hasTarget() {
		if err = st.New(bo.NewBlockOperationTargetKey(bo.Target), bo.Hash); err != nil {
			return
		}
		if err = st.New(bo.NewBlockOperationTargetAndTypeKey(bo.Target), bo.Hash); err != nil {
			return
		}
		if err = st.New(bo.NewBlockOperationPeersKey(bo.Target), bo.Hash); err != nil {
			return
		}
		if err = st.New(bo.NewBlockOperationPeersAndTypeKey(bo.Target), bo.Hash); err != nil {
			return
		}
	}

	if bo.targetIsLinked() {
		if err = st.New(GetBlockOperationCreateFrozenKey(bo.Target, bo.Height), bo.Hash); err != nil {
			return err
		}
		if err = st.New(bo.NewBlockOperationFrozenLinkedKey(bo.linked), bo.Hash); err != nil {
			return err
		}
	}

	bo.isSaved = true

	return nil
}

func key(hash string) string {
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

func keyPrefixFrozenLinked(hash string) string {
	return fmt.Sprintf(
		"%s%s",
		common.BlockOperationPrefixFrozenLinked,
		hash,
	)
}

func keyPrefixTxHash(txHash string) string {
	return fmt.Sprintf("%s%s-", common.BlockOperationPrefixTxHash, txHash)
}

func keyPrefixSource(source string) string {
	return fmt.Sprintf("%s%s-", common.BlockOperationPrefixSource, source)
}

func keyPrefixSourceAndType(source string, ty operation.OperationType) string {
	return fmt.Sprintf("%s%s%s-", common.BlockOperationPrefixTypeSource, string(ty), source)
}

func keyPrefixBlockHeight(height uint64) string {
	return fmt.Sprintf("%s%s-", common.BlockOperationPrefixBlockHeight, common.EncodeUint64ToByteSlice(height))
}

func keyPrefixTarget(target string) string {
	return fmt.Sprintf("%s%s-", common.BlockOperationPrefixTarget, target)
}

func keyPrefixTargetAndType(target string, ty operation.OperationType) string {
	return fmt.Sprintf("%s%s%s-", common.BlockOperationPrefixTypeTarget, string(ty), target)
}

func keyPrefixPeers(addr string) string {
	return fmt.Sprintf("%s%s-", common.BlockOperationPrefixPeers, addr)
}

func keyPrefixPeersAndType(addr string, ty operation.OperationType) string {
	return fmt.Sprintf("%s%s%s-", common.BlockOperationPrefixTypePeers, string(ty), addr)
}

func (bo BlockOperation) NewBlockOperationTxHashKey() string {
	return fmt.Sprintf(
		"%s%s%s%s",
		keyPrefixTxHash(bo.TxHash),
		common.EncodeUint64ToByteSlice(bo.Height),
		common.EncodeUint64ToByteSlice(bo.transaction.B.SequenceID),
		common.GetUniqueIDFromUUID(),
	)
}

func (bo BlockOperation) NewBlockOperationSourceKey() string {
	return fmt.Sprintf(
		"%s%s%s%s",
		keyPrefixSource(bo.Source),
		common.EncodeUint64ToByteSlice(bo.Height),
		common.EncodeUint64ToByteSlice(bo.transaction.B.SequenceID),
		common.GetUniqueIDFromUUID(),
	)
}

func (bo BlockOperation) NewBlockOperationFrozenLinkedKey(hash string) string {
	return fmt.Sprintf(
		"%s%s",
		keyPrefixFrozenLinked(hash),
		common.EncodeUint64ToByteSlice(bo.Height),
	)
}

func (bo BlockOperation) NewBlockOperationSourceAndTypeKey() string {
	return fmt.Sprintf(
		"%s%s%s%s",
		keyPrefixSourceAndType(bo.Source, bo.Type),
		common.EncodeUint64ToByteSlice(bo.Height),
		common.EncodeUint64ToByteSlice(bo.transaction.B.SequenceID),
		common.GetUniqueIDFromUUID(),
	)
}
func (bo BlockOperation) NewBlockOperationTargetKey(target string) string {
	return fmt.Sprintf(
		"%s%s%s%s",
		keyPrefixTarget(target),
		common.EncodeUint64ToByteSlice(bo.Height),
		common.EncodeUint64ToByteSlice(bo.transaction.B.SequenceID),
		common.GetUniqueIDFromUUID(),
	)
}

func (bo BlockOperation) NewBlockOperationTargetAndTypeKey(target string) string {
	return fmt.Sprintf(
		"%s%s%s%s",
		keyPrefixTargetAndType(target, bo.Type),
		common.EncodeUint64ToByteSlice(bo.Height),
		common.EncodeUint64ToByteSlice(bo.transaction.B.SequenceID),
		common.GetUniqueIDFromUUID(),
	)
}

func (bo BlockOperation) NewBlockOperationPeersKey(addr string) string {
	return fmt.Sprintf(
		"%s%s%s%s",
		keyPrefixPeers(addr),
		common.EncodeUint64ToByteSlice(bo.Height),
		common.EncodeUint64ToByteSlice(bo.transaction.B.SequenceID),
		common.GetUniqueIDFromUUID(),
	)
}

func (bo BlockOperation) NewBlockOperationPeersAndTypeKey(addr string) string {
	return fmt.Sprintf(
		"%s%s%s%s",
		keyPrefixPeersAndType(addr, bo.Type),
		common.EncodeUint64ToByteSlice(bo.Height),
		common.EncodeUint64ToByteSlice(bo.transaction.B.SequenceID),
		common.GetUniqueIDFromUUID(),
	)
}
func (bo BlockOperation) NewBlockOperationBlockHeightKey() string {
	return fmt.Sprintf(
		"%s%s%s",
		keyPrefixBlockHeight(bo.Height),
		common.EncodeUint64ToByteSlice(bo.transaction.B.SequenceID),
		common.GetUniqueIDFromUUID(),
	)
}

func ExistsBlockOperation(st *storage.LevelDBBackend, hash string) (bool, error) {
	return st.Has(key(hash))
}

func GetBlockOperation(st *storage.LevelDBBackend, hash string) (bo BlockOperation, err error) {
	if err = st.Get(key(hash), &bo); err != nil {
		return
	}

	bo.isSaved = true
	return
}

func GetBlockOperationWithIndex(st *storage.LevelDBBackend, hash string, opIndex int) (bo BlockOperation, err error) {
	var found = false
	iterFunc, closeFunc := GetBlockOperationsByTx(st, hash, nil)
	for idx := 0; idx <= opIndex; idx++ {

		o, hasNext, _ := iterFunc()
		if !hasNext {
			break
		}
		if idx == opIndex {
			found = true
			bo = o
			break
		}
	}
	if !found {
		err = errors.OperationNotFound
	}
	closeFunc()
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
			common.MustUnmarshalJSON(item.Value, &hash)

			bo, err := GetBlockOperation(st, hash)
			if err != nil {
				return BlockOperation{}, false, item.Key
			}

			return bo, hasNext, item.Key
		}), (func() {
			closeFunc()
		})
}

func GetBlockOperationsByTx(st *storage.LevelDBBackend, txHash string, options storage.ListOptions) (
	func() (BlockOperation, bool, []byte),
	func(),
) {
	iterFunc, closeFunc := st.GetIterator(keyPrefixTxHash(txHash), options)

	return LoadBlockOperationsInsideIterator(st, iterFunc, closeFunc)
}

func GetBlockOperationsBySource(st *storage.LevelDBBackend, source string, options storage.ListOptions) (
	func() (BlockOperation, bool, []byte),
	func(),
) {
	iterFunc, closeFunc := st.GetIterator(keyPrefixSource(source), options)

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
	iterFunc, closeFunc := st.GetIterator(keyPrefixFrozenLinked(hash), options)
	return LoadBlockOperationsInsideIterator(st, iterFunc, closeFunc)
}

func GetBlockOperationsBySourceAndType(st *storage.LevelDBBackend, source string, ty operation.OperationType, options storage.ListOptions) (
	func() (BlockOperation, bool, []byte),
	func(),
) {
	iterFunc, closeFunc := st.GetIterator(keyPrefixSourceAndType(source, ty), options)
	return LoadBlockOperationsInsideIterator(st, iterFunc, closeFunc)
}

func GetBlockOperationsByTarget(st *storage.LevelDBBackend, target string, options storage.ListOptions) (
	func() (BlockOperation, bool, []byte),
	func(),
) {
	iterFunc, closeFunc := st.GetIterator(keyPrefixTarget(target), options)

	return LoadBlockOperationsInsideIterator(st, iterFunc, closeFunc)
}

func GetBlockOperationsByTargetAndType(st *storage.LevelDBBackend, target string, ty operation.OperationType, options storage.ListOptions) (
	func() (BlockOperation, bool, []byte),
	func(),
) {
	iterFunc, closeFunc := st.GetIterator(keyPrefixTargetAndType(target, ty), options)
	return LoadBlockOperationsInsideIterator(st, iterFunc, closeFunc)
}

func GetBlockOperationsByPeers(st *storage.LevelDBBackend, addr string, options storage.ListOptions) (
	func() (BlockOperation, bool, []byte),
	func(),
) {
	iterFunc, closeFunc := st.GetIterator(keyPrefixPeers(addr), options)

	return LoadBlockOperationsInsideIterator(st, iterFunc, closeFunc)
}

func GetBlockOperationsByPeersAndType(st *storage.LevelDBBackend, addr string, ty operation.OperationType, options storage.ListOptions) (
	func() (BlockOperation, bool, []byte),
	func(),
) {
	iterFunc, closeFunc := st.GetIterator(keyPrefixPeersAndType(addr, ty), options)
	return LoadBlockOperationsInsideIterator(st, iterFunc, closeFunc)
}

func GetBlockOperationsByBlockHeight(st *storage.LevelDBBackend, height uint64, options storage.ListOptions) (
	func() (BlockOperation, bool, []byte),
	func(),
) {
	iterFunc, closeFunc := st.GetIterator(keyPrefixBlockHeight(height), options)
	return LoadBlockOperationsInsideIterator(st, iterFunc, closeFunc)
}
