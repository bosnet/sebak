package transaction

import (
	"sync"
)

type TransactionPool struct {
	sync.RWMutex

	Pool    map[ /* Transaction.GetHash() */ string]Transaction
	Sources map[ /* Transaction.Source() */ string]bool
}

func NewTransactionPool() *TransactionPool {
	return &TransactionPool{
		Pool:    map[string]Transaction{},
		Sources: map[string]bool{},
	}
}

func (tp *TransactionPool) Len() int {
	return len(tp.Pool)
}

func (tp *TransactionPool) Has(hash string) bool {
	_, found := tp.Pool[hash]
	return found
}

func (tp *TransactionPool) Get(hash string) (tx Transaction, found bool) {
	tx, found = tp.Pool[hash]
	return
}

func (tp *TransactionPool) Add(tx Transaction) bool {
	tp.RLock()
	if _, found := tp.Pool[tx.GetHash()]; found {
		return false
	}
	tp.RUnlock()

	tp.Lock()
	defer tp.Unlock()

	tp.Pool[tx.GetHash()] = tx
	tp.Sources[tx.Source()] = true

	return true
}

func (tp *TransactionPool) Remove(hashes ...string) {
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

func (tp *TransactionPool) AvailableTransactions(transactionLimit int) []string {
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

func (tp *TransactionPool) IsSameSource(source string) (found bool) {
	_, found = tp.Sources[source]

	return
}
