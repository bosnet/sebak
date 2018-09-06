package sebak

import (
	"fmt"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/error"
	"boscoin.io/sebak/lib/storage"
)

// BlockTransactionHistory is for keeping `Transaction` history. the storage should support,
//  * find by `Hash`
//  * find by `Source`
//  * sort by `Confirmed`
//  * sort by `Created`

const (
	BlockTransactionHistoryPrefixHash string = "bth-hash-" // bt-hash-<BlockTransactionHistory.Hash>
)

// TODO Is it correct to save raw `message` in BlockTransactionHistory?
// TODO Do `BlockTransactionHistory` purge the old transactions? That is, it
// just keep the recent transactions

type BlockTransactionHistory struct {
	Hash   string
	Source string

	Confirmed string
	Created   string
	Message   string

	isSaved bool
}

func NewTransactionHistoryFromTransaction(tx Transaction, message []byte) BlockTransactionHistory {
	return BlockTransactionHistory{
		Hash:      tx.H.Hash,
		Source:    tx.B.Source,
		Confirmed: sebakcommon.NowISO8601(),
		Created:   tx.H.Created,
		Message:   string(message),
	}
}

func GetBlockTransactionHistoryKey(hash string) string {
	return fmt.Sprintf("%s%s", BlockTransactionHistoryPrefixHash, hash)
}

func (bt BlockTransactionHistory) Serialize() (encoded []byte, err error) {
	encoded, err = sebakcommon.EncodeJSONValue(bt)
	return
}
func (bt *BlockTransactionHistory) Save(st *sebakstorage.LevelDBBackend) (err error) {
	if bt.isSaved {
		return errors.ErrorAlreadySaved
	}

	key := GetBlockTransactionHistoryKey(bt.Hash)

	var exists bool
	exists, err = st.Has(key)
	if err != nil {
		return
	} else if exists {
		return errors.ErrorBlockAlreadyExists
	}

	bt.Confirmed = sebakcommon.NowISO8601()
	if err = st.New(GetBlockTransactionHistoryKey(bt.Hash), bt); err != nil {
		return
	}

	bt.isSaved = true

	return nil
}

func GetBlockTransactionHistory(st *sebakstorage.LevelDBBackend, hash string) (bt BlockTransactionHistory, err error) {
	if err = st.Get(GetBlockTransactionHistoryKey(hash), &bt); err != nil {
		return
	}

	bt.isSaved = true
	return
}

func ExistsBlockTransactionHistory(st *sebakstorage.LevelDBBackend, hash string) (bool, error) {
	return st.Has(GetBlockTransactionHistoryKey(hash))
}

// BlockTransactionError stores all the non-confirmed transactions and it's reason.
// the storage should support,
//  * find by `Hash`

type BlockTransactionError struct {
	Hash string

	Reason string
}
