package block

import (
	"encoding/json"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/errors"
	"boscoin.io/sebak/lib/storage"
	"boscoin.io/sebak/lib/transaction"
	"boscoin.io/sebak/lib/transaction/operation"
)

// BlockTransaction is `Transaction` data for block. the storage should support,
//  * find by `Hash`
//
//  * get list by `SequenceID` and created order
//  * get list by `Source` and created order
//  * get list by `Confirmed` order
//  * get list by `Account` and created order
//  * get list by `Block` and created order

// TODO(BlockTransaction): support counting

type BlockTransaction struct {
	Hash  string `json:"hash"`
	Block string/* `Block.Hash` */ `json:"block"`

	SequenceID uint64        `json:"sequence_id"`
	Signature  string        `json:"signature"`
	Source     string        `json:"source"`
	Fee        common.Amount `json:"fee"`
	Operations []string      `json:"operations"`
	Amount     common.Amount `json:"amount"`

	Confirmed string `json:"confirmed"`
	Created   string `json:"created"`
	Message   []byte `json:"message"`

	Index       uint64 `json:"index"`
	BlockHeight uint64 `json:"block_height"`

	transaction transaction.Transaction
	isSaved     bool
	order       *BlockOrder
}

func NewBlockTransactionFromTransaction(blockHash string, blockHeight uint64, confirmed string, tx transaction.Transaction, index uint64) BlockTransaction {
	var opHashes []string
	for i, op := range tx.B.Operations {
		opHashes = append(opHashes, NewBlockOperationKey(op.MakeHashString(), tx.GetHash(), uint64(i)))
	}

	order := NewBlockTxOrder(blockHeight, index)

	return BlockTransaction{
		Hash:        tx.H.Hash,
		Block:       blockHash,
		SequenceID:  tx.B.SequenceID,
		Signature:   tx.H.Signature,
		Source:      tx.B.Source,
		Fee:         tx.B.Fee,
		Operations:  opHashes,
		Amount:      tx.TotalAmount(true),
		Confirmed:   confirmed,
		Created:     tx.H.Created,
		Index:       index,
		BlockHeight: blockHeight,

		transaction: tx,
		order:       order,
	}
}

func (bt BlockTransaction) NewBlockTransactionKeySource() string {
	idx := storage.NewIndex()
	idx.WritePrefix(GetBlockTransactionKeyPrefixSource(bt.Source))
	bt.order.Index(idx)
	return idx.String()

	/*
		return fmt.Sprintf(
			"%s%s%s%s",
			GetBlockTransactionKeyPrefixSource(bt.Source),
			common.EncodeUint64ToByteSlice(bt.blockHeight),
			common.EncodeUint64ToByteSlice(bt.SequenceID),
			common.EncodeUint64ToByteSlice(bt.Index),
		)
	*/
}

func (bt BlockTransaction) NewBlockTransactionKeyConfirmed() string {
	idx := storage.NewIndex()
	idx.WritePrefix(GetBlockTransactionKeyPrefixConfirmed(bt.Confirmed))
	bt.order.Index(idx)
	return idx.String()
	/*
		return fmt.Sprintf(
			"%s%s%s",
			GetBlockTransactionKeyPrefixConfirmed(bt.Confirmed),
			common.EncodeUint64ToByteSlice(bt.blockHeight),
			common.EncodeUint64ToByteSlice(bt.Index),
		)
	*/
}

func (bt BlockTransaction) NewBlockTransactionKeyHeight() string {
	idx := storage.NewIndex()
	idx.WritePrefix(common.BlockTransactionPrefixHeight)
	bt.order.Index(idx)
	return idx.String()
	/*
		return fmt.Sprintf(
			"%s%s",
			GetBlockTransactionKeyPrefixHeight(bt.blockHeight),
			common.EncodeUint64ToByteSlice(bt.Index),
		)
	*/
}

func (bt BlockTransaction) NewBlockTransactionKeyByAccount(accountAddress string) string {
	idx := storage.NewIndex()
	idx.WritePrefix(GetBlockTransactionKeyPrefixAccount(accountAddress))
	bt.order.Index(idx)
	idx.WriteOrder(common.GetUniqueIDFromUUID())
	return idx.String()
	/*
		return fmt.Sprintf(
			"%s%s%s%s",
			GetBlockTransactionKeyPrefixAccount(accountAddress),
			common.EncodeUint64ToByteSlice(bt.blockHeight),
			common.EncodeUint64ToByteSlice(bt.Index),
			common.GetUniqueIDFromUUID(),
		)
	*/
}

func (bt BlockTransaction) NewBlockTransactionKeyByBlock(hash string) string {
	idx := storage.NewIndex()
	idx.WritePrefix(GetBlockTransactionKeyPrefixBlock(hash))
	bt.order.Index(idx)
	return idx.String()
	/*
		return fmt.Sprintf(
			"%s%s%s",
			GetBlockTransactionKeyPrefixBlock(hash),
			common.EncodeUint64ToByteSlice(bt.blockHeight),
			common.EncodeUint64ToByteSlice(bt.Index),
		)
	*/
}

func (bt *BlockTransaction) Save(st *storage.LevelDBBackend) (err error) {
	if bt.isSaved {
		return errors.AlreadySaved
	}

	key := GetBlockTransactionKey(bt.Hash)

	var exists bool
	if exists, err = st.Has(key); exists || err != nil {
		if exists {
			return errors.BlockAlreadyExists
		}
		return
	}

	if err = st.New(GetBlockTransactionKey(bt.Hash), bt); err != nil {
		return
	}
	if err = st.New(bt.NewBlockTransactionKeyHeight(), bt.Hash); err != nil {
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

	bt.isSaved = true

	return nil
}

func (bt BlockTransaction) String() string {
	if encoded, err := json.Marshal(bt); err != nil {
		panic(err)
	} else {
		return string(encoded)
	}
}

func (bt BlockTransaction) Transaction() transaction.Transaction {
	if bt.transaction.IsEmpty() {
		var tx transaction.Transaction
		if len(bt.Message) < 1 {
			return tx
		}

		if err := json.Unmarshal(bt.Message, &tx); err != nil {
			return tx
		}

		bt.transaction = tx
	}

	return bt.transaction
}

func (bt *BlockTransaction) SaveBlockOperations(st *storage.LevelDBBackend) (err error) {
	if bt.Transaction().IsEmpty() {
		return errors.FailedToSaveBlockOperaton
	}

	if bt.BlockHeight < 1 {
		var blk Block
		if blk, err = GetBlock(st, bt.Block); err != nil {
			return
		} else {
			bt.BlockHeight = blk.Height
		}
	}

	for i, op := range bt.Transaction().B.Operations {
		if err = bt.SaveBlockOperation(st, op, i); err != nil {
			return
		}
	}

	return nil
}

func (bt *BlockTransaction) SaveBlockOperation(st *storage.LevelDBBackend, op operation.Operation, opIndex int) (err error) {
	if bt.BlockHeight < 1 {
		var blk Block
		if blk, err = GetBlock(st, bt.Block); err != nil {
			return
		} else {
			bt.BlockHeight = blk.Height
		}
	}

	var bo BlockOperation
	bo, err = NewBlockOperationFromOperation(op, bt.Transaction(), bt.BlockHeight, bt.Index, opIndex)
	if err != nil {
		return
	}
	if err = bo.Save(st); err != nil {
		return
	}
	if pop, ok := op.B.(operation.Payable); ok {
		err = st.New(bt.NewBlockTransactionKeyByAccount(pop.TargetAddress()), bt.Hash)
		if err != nil {
			return
		}
	}

	return nil
}

//TODO: This function is no longer required when Index for operation is applied
func (bt *BlockTransaction) GetOperationIndex(opHash string) (opIndex int, err error) {
	opIndex = -1
	for i, op := range bt.Operations {
		if op == opHash {
			opIndex = i
		}
	}
	if opIndex <= -1 {
		err = errors.OperationNotFound
	}
	return
}

func (bt BlockTransaction) BlockOrder() *BlockOrder {
	return bt.order
}

func GetBlockTransactionKeyPrefixSource(source string) string {
	idx := storage.NewIndex()
	return idx.WritePrefix(common.BlockTransactionPrefixSource, source).String()
	//return fmt.Sprintf("%s%s-", common.BlockTransactionPrefixSource, source)
}

func GetBlockTransactionKeyPrefixConfirmed(confirmed string) string {
	idx := storage.NewIndex()
	return idx.WritePrefix(common.BlockTransactionPrefixConfirmed, confirmed).String()
	//return fmt.Sprintf("%s%s-", common.BlockTransactionPrefixConfirmed, confirmed)
}

func GetBlockTransactionKeyPrefixAccount(accountAddress string) string {
	idx := storage.NewIndex()
	return idx.WritePrefix(common.BlockTransactionPrefixAccount, accountAddress).String()
	//return fmt.Sprintf("%s%s-", common.BlockTransactionPrefixAccount, accountAddress)
}

func GetBlockTransactionKeyPrefixBlock(hash string) string {
	idx := storage.NewIndex()
	return idx.WritePrefix(common.BlockTransactionPrefixBlock, hash).String()
	//return fmt.Sprintf("%s%s-", common.BlockTransactionPrefixBlock, hash)
}

func GetBlockTransactionKey(hash string) string {
	idx := storage.NewIndex()
	return idx.WritePrefix(common.BlockTransactionPrefixHash, hash).String()
	//return fmt.Sprintf("%s%s-", common.BlockTransactionPrefixHash, hash)
}

func GetBlockTransactionKeyPrefixHeight(height uint64) string {
	idx := storage.NewIndex()
	idx.WritePrefix(
		common.BlockTransactionPrefixHeight,
	)
	return idx.WriteOrder(common.EncodeUint64ToString(height)).String()
	//return fmt.Sprintf("%s%s-", common.BlockTransactionPrefixHeight, common.EncodeUint64ToByteSlice(height))
}

func GetBlockTransaction(st *storage.LevelDBBackend, hash string) (bt BlockTransaction, err error) {
	if err = st.Get(GetBlockTransactionKey(hash), &bt); err != nil {
		return
	}

	bt.isSaved = true
	bt.order = NewBlockTxOrder(bt.BlockHeight, bt.Index)
	return
}

func ExistsBlockTransaction(st *storage.LevelDBBackend, hash string) (bool, error) {
	return st.Has(GetBlockTransactionKey(hash))
}

func LoadBlockTransactionsInsideIterator(
	st *storage.LevelDBBackend,
	iterFunc func() (storage.IterItem, bool),
	closeFunc func(),
) (
	func() (BlockTransaction, bool, []byte),
	func(),
) {

	return (func() (BlockTransaction, bool, []byte) {
			item, hasNext := iterFunc()
			if !hasNext {
				return BlockTransaction{}, false, item.Key
			}

			var hash string
			common.MustUnmarshalJSON(item.Value, &hash)

			bt, err := GetBlockTransaction(st, hash)
			if err != nil {
				return BlockTransaction{}, false, item.Key
			}

			return bt, hasNext, item.Key
		}), (func() {
			closeFunc()
		})
}

func GetBlockTransactionsBySource(st *storage.LevelDBBackend, source string, options storage.ListOptions) (
	func() (BlockTransaction, bool, []byte),
	func(),
) {
	iterFunc, closeFunc := st.GetIterator(GetBlockTransactionKeyPrefixSource(source), options)

	return LoadBlockTransactionsInsideIterator(st, iterFunc, closeFunc)
}

func GetBlockTransactionsByConfirmed(st *storage.LevelDBBackend, options storage.ListOptions) (
	func() (BlockTransaction, bool, []byte),
	func(),
) {
	iterFunc, closeFunc := st.GetIterator(common.BlockTransactionPrefixConfirmed, options)

	return LoadBlockTransactionsInsideIterator(st, iterFunc, closeFunc)
}

func GetBlockTransactionsByAccount(st *storage.LevelDBBackend, accountAddress string, options storage.ListOptions) (
	func() (BlockTransaction, bool, []byte),
	func(),
) {
	iterFunc, closeFunc := st.GetIterator(GetBlockTransactionKeyPrefixAccount(accountAddress), options)
	return LoadBlockTransactionsInsideIterator(st, iterFunc, closeFunc)
}

func GetBlockTransactionsByBlock(st *storage.LevelDBBackend, hash string, options storage.ListOptions) (
	func() (BlockTransaction, bool, []byte),
	func(),
) {
	iterFunc, closeFunc := st.GetIterator(GetBlockTransactionKeyPrefixBlock(hash), options)
	return LoadBlockTransactionsInsideIterator(st, iterFunc, closeFunc)
}

func GetBlockTransactionsByHeight(st *storage.LevelDBBackend, height uint64, options storage.ListOptions) (
	func() (BlockTransaction, bool, []byte),
	func(),
) {
	iterFunc, closeFunc := st.GetIterator(GetBlockTransactionKeyPrefixHeight(height), options)
	return LoadBlockTransactionsInsideIterator(st, iterFunc, closeFunc)
}

func GetBlockTransactions(st *storage.LevelDBBackend, options storage.ListOptions) (
	func() (BlockTransaction, bool, []byte),
	func(),
) {
	iterFunc, closeFunc := st.GetIterator(common.BlockTransactionPrefixHeight, options)
	return LoadBlockTransactionsInsideIterator(st, iterFunc, closeFunc)
}
