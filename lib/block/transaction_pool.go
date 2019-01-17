package block

import (
	"encoding/json"
	"fmt"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/common/observer"
	"boscoin.io/sebak/lib/errors"
	"boscoin.io/sebak/lib/storage"
	"boscoin.io/sebak/lib/transaction"
)

// TransactionPool is not `transaction.Pool`; this contains the all the valid
// transactions. The unconfirmed transactions of `TransactionPool` can be
// removed, but confirmed transactions must be here.
//
// NOTE The sync process must store the confirmed transactions.
type TransactionPool struct {
	Hash    string `json:"hash"`
	Message []byte `json:"message"`

	transaction transaction.Transaction
}

func NewTransactionPool(tx transaction.Transaction) (tp TransactionPool, err error) {
	var b []byte
	if b, err = json.Marshal(tx); err != nil {
		return
	}

	return TransactionPool{
		Hash:    tx.GetHash(),
		Message: b,
	}, nil
}

func GetTransactionPoolKey(hash string) string {
	return fmt.Sprintf("%s%s", common.TransactionPoolPrefix, hash)
}

func (tp TransactionPool) Save(st *storage.LevelDBBackend) (err error) {
	key := GetTransactionPoolKey(tp.Hash)

	var exists bool
	if exists, err = st.Has(key); exists || err != nil {
		if exists {
			return errors.BlockAlreadyExists
		}
		return
	}

	if err = st.New(key, tp); err != nil {
		return
	}

	event := observer.NewCondition(observer.TxPool, observer.Identifier, tp.Hash).String()
	go observer.ResourceObserver.Trigger(event, &tp)

	return nil
}

func (tp TransactionPool) Serialize() ([]byte, error) {
	return json.Marshal(tp)
}

func (tp TransactionPool) String() string {
	if encoded, err := json.Marshal(tp); err != nil {
		panic(err)
	} else {
		return string(encoded)
	}
}

func (tp TransactionPool) Transaction() transaction.Transaction {
	if tp.transaction.IsEmpty() {
		var tx transaction.Transaction
		if err := json.Unmarshal(tp.Message, &tx); err != nil {
			return tx
		}

		tp.transaction = tx
	}

	return tp.transaction
}

func ExistsTransactionPool(st *storage.LevelDBBackend, hash string) (bool, error) {
	return st.Has(GetTransactionPoolKey(hash))
}

func GetTransactionPool(st *storage.LevelDBBackend, hash string) (tp TransactionPool, err error) {
	err = st.Get(GetTransactionPoolKey(hash), &tp)
	return
}

func DeleteTransactionPool(st *storage.LevelDBBackend, hash string) error {
	return st.Remove(GetTransactionPoolKey(hash))
}

func SaveTransactionPool(st *storage.LevelDBBackend, tx transaction.Transaction) (tp TransactionPool, err error) {
	if tp, err = NewTransactionPool(tx); err != nil {
		return
	}

	err = tp.Save(st)
	if err == nil {
		return
	}

	if err == errors.BlockAlreadyExists {
		err = nil
		return
	}

	return
}
