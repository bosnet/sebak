package sebak

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/spikeekips/sebak/lib/error"
	"github.com/spikeekips/sebak/lib/storage"
	"github.com/spikeekips/sebak/lib/util"
)

/*
BlockOperation is `Operation` data for block. the storage should support,
 * find by `Hash`
 * find by `TxHash`

 * get list by `Source` and created order
 * get list by `Target` and created order

*/

const (
	BlockOperationPrefixHash   string = "bo-hash-"   // bo-hash-<BlockOperation.Hash>
	BlockOperationPrefixTxHash string = "bo-txhash-" // bo-txhash-<BlockOperation.TxHash>-<created>
	BlockOperationPrefixSource string = "bo-source-" // bo-source-<BlockOperation.Source>-<created>
	BlockOperationPrefixTarget string = "bo-target-" // bo-target-<BlockOperation.Target>-<created>
	BlockOperationPrefixPeers  string = "bo-peers-"  // bo-target-<Address0>-<Address1>-<created>
)

type BlockOperation struct {
	Hash   string
	TxHash string

	Type   OperationType
	Source string
	Target string
	Amount Amount
}

func NewBlockOperationFromOperation(op Operation, tx Transaction) BlockOperation {
	return BlockOperation{
		Hash:   op.MakeHashString(),
		TxHash: tx.H.Hash,

		Type:   op.H.Type,
		Source: tx.B.Source,
		Target: op.B.TargetAddress(),
		Amount: op.B.GetAmount(),
	}
}

func (bo BlockOperation) Save(st *storage.LevelDBBackend) (err error) {
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

	return nil
}

func (bo BlockOperation) Serialize() (encoded []byte, err error) {
	encoded, err = util.EncodeJSONValue(bo)
	return
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

func GetBlockOperationKeyPrefixPeers(one, two string) string {
	return fmt.Sprintf("%s%s%s-", BlockOperationPrefixPeers, one, two)
}

func (bo BlockOperation) NewBlockOperationTxHashKey() string {
	return fmt.Sprintf(
		"%s%s",
		GetBlockOperationKeyPrefixTxHash(bo.TxHash),
		util.GetUniqueIDFromUUID(),
	)
}

func (bo BlockOperation) NewBlockOperationSourceKey() string {
	return fmt.Sprintf(
		"%s%s",
		GetBlockOperationKeyPrefixSource(bo.Source),
		util.GetUniqueIDFromUUID(),
	)
}

func (bo BlockOperation) NewBlockOperationTargetKey() string {
	return fmt.Sprintf(
		"%s%s",
		GetBlockOperationKeyPrefixTarget(bo.Target),
		util.GetUniqueIDFromUUID(),
	)
}

func (bo BlockOperation) NewBlockOperationPeersKey() string {
	addresses := []string{bo.Target, bo.Source}
	sort.Strings(addresses)
	return fmt.Sprintf(
		"%s%s",
		GetBlockOperationKeyPrefixPeers(addresses[0], addresses[1]),
		util.GetUniqueIDFromUUID(),
	)
}

func ExistBlockOperation(st *storage.LevelDBBackend, hash string) (bool, error) {
	return st.Has(GetBlockOperationKey(hash))
}

func GetBlockOperation(st *storage.LevelDBBackend, hash string) (bo BlockOperation, err error) {
	if err = st.Get(GetBlockOperationKey(hash), &bo); err != nil {
		return
	}

	return
}

func LoadBlockOperationsInsideIterator(
	st *storage.LevelDBBackend,
	iterFunc func() (storage.IterItem, bool),
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

func GetBlockOperationsByTxHash(st *storage.LevelDBBackend, txHash string, reverse bool) (
	func() (BlockOperation, bool),
	func(),
) {
	iterFunc, closeFunc := st.GetIterator(GetBlockOperationKeyPrefixTxHash(txHash), reverse)

	return LoadBlockOperationsInsideIterator(st, iterFunc, closeFunc)
}

func GetBlockOperationsBySource(st *storage.LevelDBBackend, source string, reverse bool) (
	func() (BlockOperation, bool),
	func(),
) {
	iterFunc, closeFunc := st.GetIterator(GetBlockOperationKeyPrefixSource(source), reverse)

	return LoadBlockOperationsInsideIterator(st, iterFunc, closeFunc)
}

func GetBlockOperationsByTarget(st *storage.LevelDBBackend, target string, reverse bool) (
	func() (BlockOperation, bool),
	func(),
) {
	iterFunc, closeFunc := st.GetIterator(GetBlockOperationKeyPrefixTarget(target), reverse)

	return LoadBlockOperationsInsideIterator(st, iterFunc, closeFunc)
}
