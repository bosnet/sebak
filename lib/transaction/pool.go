package transaction

import (
	"sync"
)

type Pool struct {
	sync.RWMutex

	Pool    map[ /* Transaction.GetHash() */ string]Transaction
	Sources map[ /* Transaction.Source() */ string]bool
}

func NewPool() *Pool {
	return &Pool{
		Pool:    map[string]Transaction{},
		Sources: map[string]bool{},
	}
}

func (tp *Pool) Len() int {
	return len(tp.Pool)
}

func (tp *Pool) Has(hash string) bool {
	tp.RLock()
	defer tp.RUnlock()
	_, found := tp.Pool[hash]
	return found
}

func (tp *Pool) Get(hash string) (tx Transaction, found bool) {
	tx, found = tp.Pool[hash]
	return
}

func (tp *Pool) Add(tx Transaction) bool {
	if tp.Has(tx.GetHash()) {
		return false
	}

	tp.Lock()
	defer tp.Unlock()

	tp.Pool[tx.GetHash()] = tx
	tp.Sources[tx.Source()] = true

	return true
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
	_, found = tp.Sources[source]

	return
}
