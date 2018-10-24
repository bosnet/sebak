package transaction

import (
	"sync"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/error"
)

type Pool struct {
	sync.RWMutex

	Pool    map[ /* Transaction.GetHash() */ string]Transaction
	Sources map[ /* Transaction.Source() */ string]bool

	limit int
}

func NewPool(limit int) *Pool {
	if limit <= 0 {
		limit = common.DefaultTxPoolLimit
	}
	return &Pool{
		Pool:    map[string]Transaction{},
		Sources: map[string]bool{},
		limit:   limit,
	}
}

func (tp *Pool) Len() int {
	tp.RLock()
	defer tp.RUnlock()

	return len(tp.Pool)
}

func (tp *Pool) Has(hash string) bool {
	tp.RLock()
	defer tp.RUnlock()

	_, found := tp.Pool[hash]
	return found
}

func (tp *Pool) Get(hash string) (tx Transaction, found bool) {
	tp.RLock()
	defer tp.RUnlock()

	tx, found = tp.Pool[hash]
	return
}

func (tp *Pool) Add(tx Transaction) error {
	if tp.Has(tx.GetHash()) {
		return errors.ErrorTransactionAlreadyExistsInPool
	}

	tp.Lock()
	defer tp.Unlock()

	if len(tp.Pool) >= tp.limit {
		return errors.ErrorTransactionPoolFull
	}

	tp.Pool[tx.GetHash()] = tx
	tp.Sources[tx.Source()] = true

	return nil
}

func (tp *Pool) Remove(hashes ...string) {
	if len(hashes) < 1 {
		return
	}

	tp.Lock()
	defer tp.Unlock()

	for _, hash := range hashes {
		if tx, found := tp.Pool[hash]; found {
			delete(tp.Sources, tx.Source())
			delete(tp.Pool, hash)
		}
	}
}

func (tp *Pool) AvailableTransactions(transactionLimit int) []string {
	if transactionLimit < 1 {
		return nil
	}

	tp.RLock()
	defer tp.RUnlock()

	var ret []string
	for key, _ := range tp.Pool {
		if len(ret) == transactionLimit {
			return ret
		}
		ret = append(ret, key)
	}
	return ret
}

func (tp *Pool) IsSameSource(source string) (found bool) {
	tp.RLock()
	defer tp.RUnlock()

	_, found = tp.Sources[source]

	return
}
