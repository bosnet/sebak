package sebak

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/spikeekips/sebak/lib/common"
	"github.com/spikeekips/sebak/lib/error"
	"github.com/spikeekips/sebak/lib/storage"
)

/*
BlockOperation is `Operation` data for block. the storage should support,
 * find by `Hash`
 * find by `TxHash`

 * get list by `Source` and created order
 * get list by `Target` and created order

*/

const (
	BlockOperationPrefixHash       string = "bo-hash-"       // bo-hash-<BlockOperation.Hash>
	BlockOperationPrefixTxHash     string = "bo-txhash-"     // bo-txhash-<BlockOperation.TxHash>-<created>
	BlockOperationPrefixSource     string = "bo-source-"     // bo-source-<BlockOperation.Source>-<created>
	BlockOperationPrefixTarget     string = "bo-target-"     // bo-target-<BlockOperation.Target>-<created>
	BlockOperationPrefixPeers      string = "bo-peers-"      // bo-target-<Address0>-<Address1>-<created>
	BlockOperationPrefixCheckpoint string = "bo-checkpoint-" // bo-checkpoint-<Transaction.B.Checkpoint>-<created>
)

type BlockOperation struct {
	Hash   string
	TxHash string

	Type   OperationType
	Source string
	Target string
	Amount Amount

	// transaction will be used only for `Save` time.
	transaction Transaction
	isSaved     bool
}

func NewBlockOperationFromOperation(op Operation, tx Transaction) BlockOperation {
	return BlockOperation{
		Hash:   op.MakeHashString(),
		TxHash: tx.H.Hash,

		Type:   op.H.Type,
		Source: tx.B.Source,
		Target: op.B.TargetAddress(),
		Amount: op.B.GetAmount(),

		transaction: tx,
	}
}

func (bo *BlockOperation) Save(st *sebakstorage.LevelDBBackend) (err error) {
	if bo.isSaved {
		return sebakerror.ErrorAlreadySaved
	}

	key := GetBlockOperationKey(bo.Hash)

	var exists bool
	if exists, err = st.Has(key); err != nil {
		return
	} else if exists {
		return sebakerror.ErrorBlockAlreadyExists
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
	if err = st.New(bo.NewBlockOperationCheckpoint(), bo.Hash); err != nil {
		return
	}

	bo.isSaved = true

	return nil
}

func (bo BlockOperation) Serialize() (encoded []byte, err error) {
	encoded, err = sebakcommon.EncodeJSONValue(bo)
	return
}

func (bo BlockOperation) Transaction() Transaction {
	return bo.transaction
}

func GetBlockOperationKey(hash string) string {
	return fmt.Sprintf("%s%s", BlockOperationPrefixHash, hash)
}

func GetBlockOperationKeyPrefixTxHash(txHash string) string {
	return fmt.Sprintf("%s%s-", BlockOperationPrefixTxHash, txHash)
}

func GetBlockOperationKeyPrefixSource(source string) string {
	return fmt.Sprintf("%s%s-", BlockOperationPrefixSource, source)
}

func GetBlockOperationKeyPrefixTarget(target string) string {
	return fmt.Sprintf("%s%s-", BlockOperationPrefixTarget, target)
}

func GetBlockOperationKeyPrefixCheckpoint(checkpoint string) string {
	return fmt.Sprintf("%s%s-", BlockOperationPrefixTarget, checkpoint)
}

func GetBlockOperationKeyPrefixPeers(one, two string) string {
	return fmt.Sprintf("%s%s%s-", BlockOperationPrefixPeers, one, two)
}

func (bo BlockOperation) NewBlockOperationTxHashKey() string {
	return fmt.Sprintf(
		"%s%s",
		GetBlockOperationKeyPrefixTxHash(bo.TxHash),
		sebakcommon.GetUniqueIDFromUUID(),
	)
}

func (bo BlockOperation) NewBlockOperationSourceKey() string {
	return fmt.Sprintf(
		"%s%s",
		GetBlockOperationKeyPrefixSource(bo.Source),
		sebakcommon.GetUniqueIDFromUUID(),
	)
}

func (bo BlockOperation) NewBlockOperationTargetKey() string {
	return fmt.Sprintf(
		"%s%s",
		GetBlockOperationKeyPrefixTarget(bo.Target),
		sebakcommon.GetUniqueIDFromUUID(),
	)
}

func (bo BlockOperation) NewBlockOperationCheckpoint() string {
	return fmt.Sprintf(
		"%s%s",
		GetBlockOperationKeyPrefixTarget(bo.transaction.B.Checkpoint),
		sebakcommon.GetUniqueIDFromUUID(),
	)
}

func (bo BlockOperation) NewBlockOperationPeersKey() string {
	addresses := []string{bo.Target, bo.Source}
	sort.Strings(addresses)
	return fmt.Sprintf(
		"%s%s",
		GetBlockOperationKeyPrefixPeers(addresses[0], addresses[1]),
		sebakcommon.GetUniqueIDFromUUID(),
	)
}

func ExistBlockOperation(st *sebakstorage.LevelDBBackend, hash string) (bool, error) {
	return st.Has(GetBlockOperationKey(hash))
}

func GetBlockOperation(st *sebakstorage.LevelDBBackend, hash string) (bo BlockOperation, err error) {
	if err = st.Get(GetBlockOperationKey(hash), &bo); err != nil {
		return
	}

	bo.isSaved = true
	return
}

func LoadBlockOperationsInsideIterator(
	st *sebakstorage.LevelDBBackend,
	iterFunc func() (sebakstorage.IterItem, bool),
	closeFunc func(),
) (
	func() (BlockOperation, bool),
	func(),
) {

	return (func() (BlockOperation, bool) {
			item, hasNext := iterFunc()
			if !hasNext {
				return BlockOperation{}, false
			}

			var hash string
			json.Unmarshal(item.Value, &hash)

			bo, err := GetBlockOperation(st, hash)
			if err != nil {
				return BlockOperation{}, false
			}

			return bo, hasNext
		}), (func() {
			closeFunc()
		})
}

func GetBlockOperationsByTxHash(st *sebakstorage.LevelDBBackend, txHash string, reverse bool) (
	func() (BlockOperation, bool),
	func(),
) {
	iterFunc, closeFunc := st.GetIterator(GetBlockOperationKeyPrefixTxHash(txHash), reverse)

	return LoadBlockOperationsInsideIterator(st, iterFunc, closeFunc)
}

func GetBlockOperationsBySource(st *sebakstorage.LevelDBBackend, source string, reverse bool) (
	func() (BlockOperation, bool),
	func(),
) {
	iterFunc, closeFunc := st.GetIterator(GetBlockOperationKeyPrefixSource(source), reverse)

	return LoadBlockOperationsInsideIterator(st, iterFunc, closeFunc)
}

func GetBlockOperationsByTarget(st *sebakstorage.LevelDBBackend, target string, reverse bool) (
	func() (BlockOperation, bool),
	func(),
) {
	iterFunc, closeFunc := st.GetIterator(GetBlockOperationKeyPrefixTarget(target), reverse)

	return LoadBlockOperationsInsideIterator(st, iterFunc, closeFunc)
}

func GetBlockOperationsByCheckpoint(st *sebakstorage.LevelDBBackend, checkpoint string, reverse bool) (
	func() (BlockOperation, bool),
	func(),
) {
	iterFunc, closeFunc := st.GetIterator(GetBlockOperationKeyPrefixCheckpoint(checkpoint), reverse)

	return LoadBlockOperationsInsideIterator(st, iterFunc, closeFunc)
}
