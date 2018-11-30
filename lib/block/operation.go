package block

import (
	"encoding/json"
	"fmt"
	"strconv"

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

	Type    operation.OperationType `json:"type"`
	Source  string                  `json:"source"`
	Target  string                  `json:"target"`
	Body    []byte                  `json:"body"`
	Height  uint64                  `json:"block_height"`
	Index   uint64                  `json:"index"`
	TxIndex uint64                  `json:"tx_index"`

	// bellows will be used only for `Save` time.
	transaction transaction.Transaction
	operation   operation.Operation
	linked      string
	isSaved     bool
	order       *BlockOrder
}

func NewBlockOperationKey(opHash, txHash string, index uint64) string {
	return common.MustMakeObjectHashString([]string{opHash, txHash, strconv.FormatUint(index, 10)})
}

func NewBlockOperationFromOperation(op operation.Operation, tx transaction.Transaction, blockHeight uint64, txIndex uint64, opIndex int) (BlockOperation, error) {
	body, err := json.Marshal(op.B)
	if err != nil {
		return BlockOperation{}, err
	}

	opHash := op.MakeHashString()
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
	order := NewBlockOpOrder(blockHeight, txIndex, uint64(opIndex))

	return BlockOperation{
		Hash: NewBlockOperationKey(opHash, txHash, uint64(opIndex)),

		OpHash: opHash,
		TxHash: txHash,

		Type:    op.H.Type,
		Source:  tx.B.Source,
		Target:  target,
		Body:    body,
		Height:  blockHeight,
		Index:   uint64(opIndex),
		TxIndex: uint64(txIndex),

		transaction: tx,
		operation:   op,
		linked:      linked,
		order:       order,
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
	if err = st.New(bo.NewBlockOperationPeersAndTypeKey(bo.Source, bo.Type), bo.Hash); err != nil {
		return
	}
	if err = st.New(bo.NewBlockOperationBlockHeightKey(), bo.Hash); err != nil {
		return
	}

	if bo.hasTarget() {
		if err = st.New(bo.NewBlockOperationTargetKey(bo.Target), bo.Hash); err != nil {
			return
		}
		if err = st.New(bo.NewBlockOperationTargetAndTypeKey(bo.Target, bo.Type), bo.Hash); err != nil {
			return
		}
		if err = st.New(bo.NewBlockOperationPeersKey(bo.Target), bo.Hash); err != nil {
			return
		}
		if err = st.New(bo.NewBlockOperationPeersAndTypeKey(bo.Target, bo.Type), bo.Hash); err != nil {
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
	return fmt.Sprintf("%s%s-", common.BlockOperationPrefixBlockHeight, common.EncodeUint64ToString(height))
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

func GetBlockOperationKey(hash string) string {
	idx := storage.NewIndex()
	idx.WritePrefix(common.BlockOperationPrefixHash, hash)
	return idx.String()
}

func GetBlockOperationCreateFrozenKey(hash string, height uint64) string {
	idx := storage.NewIndex()
	idx.WritePrefix(common.BlockOperationPrefixCreateFrozen)
	idx.WritePrefix(common.EncodeUint64ToString(height))
	idx.WritePrefix(hash)
	return idx.String()
}

func GetBlockOperationKeyPrefixFrozenLinked(hash string) string {
	idx := storage.NewIndex()
	return idx.WritePrefix(common.BlockOperationPrefixFrozenLinked, hash).String()
}

func GetBlockOperationKeyPrefixTxHash(txHash string) string {
	idx := storage.NewIndex()
	idx.WritePrefix(common.BlockOperationPrefixTxHash, txHash)
	return idx.String()
}

func GetBlockOperationKeyPrefixSource(source string) string {
	idx := storage.NewIndex()
	idx.WritePrefix(common.BlockOperationPrefixSource, source)
	return idx.String()
}

func GetBlockOperationKeyPrefixSourceAndType(source string, ty operation.OperationType) string {
	idx := storage.NewIndex()
	idx.WritePrefix(common.BlockOperationPrefixTypeSource, string(ty), source)
	return idx.String()
}

func GetBlockOperationKeyPrefixBlockHeight(height uint64) string {
	idx := storage.NewIndex()
	idx.WritePrefix(common.BlockOperationPrefixBlockHeight)
	idx.WritePrefix(common.EncodeUint64ToString(height))
	return idx.String()
}

func GetBlockOperationKeyPrefixTarget(target string) string {
	idx := storage.NewIndex()
	idx.WritePrefix(common.BlockOperationPrefixTarget, target)
	return idx.String()
}

func GetBlockOperationKeyPrefixTargetAndType(target string, ty operation.OperationType) string {
	idx := storage.NewIndex()
	idx.WritePrefix(common.BlockOperationPrefixTypeTarget)
	idx.WritePrefix(string(ty), target)
	return idx.String()
}

func GetBlockOperationKeyPrefixPeers(addr string) string {
	idx := storage.NewIndex()
	idx.WritePrefix(common.BlockOperationPrefixPeers, addr)
	return idx.String()
}

func GetBlockOperationKeyPrefixPeersAndType(addr string, ty operation.OperationType) string {
	idx := storage.NewIndex()
	idx.WritePrefix(common.BlockOperationPrefixTypePeers)
	idx.WritePrefix(string(ty), addr)
	return idx.String()
}

func (bo BlockOperation) NewBlockOperationTxHashKey() string {
	idx := storage.NewIndex()
	idx.WritePrefix(GetBlockOperationKeyPrefixTxHash(bo.TxHash))
	bo.order.Index(idx)
	return idx.String()
}

func (bo BlockOperation) NewBlockOperationSourceKey() string {
	idx := storage.NewIndex()
	idx.WritePrefix(GetBlockOperationKeyPrefixSource(bo.Source))
	bo.order.Index(idx)
	return idx.String()
}

func (bo BlockOperation) NewBlockOperationFrozenLinkedKey(hash string) string {
	idx := storage.NewIndex()
	idx.WritePrefix(GetBlockOperationKeyPrefixFrozenLinked(hash))
	bo.order.Index(idx)
	return idx.String()
}

func (bo BlockOperation) NewBlockOperationSourceAndTypeKey() string {
	idx := storage.NewIndex()
	idx.WritePrefix(GetBlockOperationKeyPrefixSourceAndType(bo.Source, bo.Type))
	bo.order.Index(idx)
	return idx.String()
}

func (bo BlockOperation) NewBlockOperationTargetKey(target string) string {
	idx := storage.NewIndex()
	idx.WritePrefix(GetBlockOperationKeyPrefixTarget(target))
	bo.order.Index(idx)
	return idx.String()
}

func (bo BlockOperation) NewBlockOperationTargetAndTypeKey(target string, ty operation.OperationType) string {
	idx := storage.NewIndex()
	idx.WritePrefix(GetBlockOperationKeyPrefixTargetAndType(target, ty))
	bo.order.Index(idx)
	return idx.String()
}

func (bo BlockOperation) NewBlockOperationPeersKey(addr string) string {
	idx := storage.NewIndex()
	idx.WritePrefix(GetBlockOperationKeyPrefixPeers(addr))
	bo.order.Index(idx)
	return idx.String()
}

func (bo BlockOperation) NewBlockOperationPeersAndTypeKey(addr string, ty operation.OperationType) string {
	idx := storage.NewIndex()
	idx.WritePrefix(GetBlockOperationKeyPrefixPeersAndType(addr, ty))
	bo.order.Index(idx)
	return idx.String()
}

func (bo BlockOperation) NewBlockOperationBlockHeightKey() string {
	idx := storage.NewIndex()
	idx.WritePrefix(GetBlockOperationKeyPrefixBlockHeight(bo.Height))
	bo.order.Index(idx)
	return idx.String()
}

func (bo BlockOperation) BlockOrder() *BlockOrder {
	return bo.order
}

func ExistsBlockOperation(st *storage.LevelDBBackend, hash string) (bool, error) {
	return st.Has(key(hash))
}

func GetBlockOperation(st *storage.LevelDBBackend, hash string) (bo BlockOperation, err error) {
	if err = st.Get(key(hash), &bo); err != nil {
		return
	}

	bo.isSaved = true
	bo.order = NewBlockOpOrder(bo.Height, bo.TxIndex, bo.Index)
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
			if !hasNext && (item.Key == nil || item.Value == nil) {
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
