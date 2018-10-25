package block

import (
	"fmt"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/common/observer"
	"boscoin.io/sebak/lib/storage"
	"boscoin.io/sebak/lib/transaction"
)

// BlockTransactionHistory is for keeping `Transaction` history. the storage should support,
//  * find by `Hash`
//  * find by `Source`
//  * sort by `Confirmed`
//  * sort by `Created`

const (
	BlockTransactionHistoryPrefixHash string = "bth-hash-" // bt-hash-<BlockTransactionHistory.Hash>
)

const (
	TransactionHistoryStatusSubmitted = "submitted"
	TransactionHistoryStatusConfirmed = "confirmed"
	TransactionHistoryStatusRejected  = "rejected"
)

type BlockTransactionHistory struct {
	Hash   string `json:"hash"`
	Source string `json:"source"`

	Time    string `json:"time"`
	Created string `json:"created"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

func NewTransactionHistoryFromTransaction(tx transaction.Transaction, message []byte) BlockTransactionHistory {
	return BlockTransactionHistory{
		Hash:    tx.H.Hash,
		Source:  tx.B.Source,
		Created: tx.H.Created,
		Message: string(message),
	}
}

func SaveTransactionHistory(st *storage.LevelDBBackend, tx transaction.Transaction, message []byte, status string) (err error) {
	bth, err := GetBlockTransactionHistory(st, tx.H.Hash)
	if err != nil {
		bth = NewTransactionHistoryFromTransaction(tx, message)
	}
	if bth.Status != "" && bth.Status != TransactionHistoryStatusSubmitted {
		// Only TransactionHistoryStatusSubmitted can be changed
		return nil
	}
	bth.Status = status
	return bth.Save(st)
}

func GetBlockTransactionHistoryKey(hash string) string {
	return fmt.Sprintf("%s%s", BlockTransactionHistoryPrefixHash, hash)
}

func (bt BlockTransactionHistory) Serialize() (encoded []byte, err error) {
	encoded, err = common.EncodeJSONValue(bt)
	return
}
func (bt *BlockTransactionHistory) Save(st *storage.LevelDBBackend) (err error) {

	key := GetBlockTransactionHistoryKey(bt.Hash)

	var exists bool
	exists, err = st.Has(key)
	if err != nil {
		return
	}

	bt.Time = common.NowISO8601()
	if exists {
		err = st.Set(key, bt)
	} else {
		err = st.New(GetBlockTransactionHistoryKey(bt.Hash), bt)
	}

	event := "saved"
	event += " " + fmt.Sprintf("hash-%s", bt.Hash)
	observer.BlockTransactionHistoryObserver.Trigger(event, bt)
	return nil
}

func GetBlockTransactionHistory(st *storage.LevelDBBackend, hash string) (bt BlockTransactionHistory, err error) {
	if err = st.Get(GetBlockTransactionHistoryKey(hash), &bt); err != nil {
		return
	}

	return
}

func ExistsBlockTransactionHistory(st *storage.LevelDBBackend, hash string) (bool, error) {
	return st.Has(GetBlockTransactionHistoryKey(hash))
}

// BlockTransactionError stores all the non-confirmed transactions and it's reason.
// the storage should support,
//  * find by `Hash`

type BlockTransactionError struct {
	Hash string

	Reason string
}
