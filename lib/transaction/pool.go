package transaction

import (
	"container/list"
	"sync"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/errors"
	"boscoin.io/sebak/lib/metrics"
)

type Pool struct {
	sync.RWMutex

	Pool    map[ /* Transaction.GetHash() */ string]Transaction
	sources map[ /* Transaction.Source() */ string] /* Transaction.GetHash() */ string

	hashList *list.List // Transaction.GetHash()
	hashMap  map[ /* Transaction.GetHash() */ string]*list.Element

	limit int
}

func NewPool(limit int) *Pool {
	if limit <= 0 {
		limit = common.DefaultTxPoolLimit
	}
	return &Pool{
		Pool:     map[string]Transaction{},
		sources:  map[string]string{},
		hashList: list.New(),
		hashMap:  make(map[string]*list.Element),
		limit:    limit,
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

func (tp *Pool) Get(hash string) (Transaction, bool) {
	tp.RLock()
	defer tp.RUnlock()

	tx, found := tp.Pool[hash]
	return tx, found
}

func (tp *Pool) GetFromSource(source string) (Transaction, bool) {
	tp.RLock()
	defer tp.RUnlock()
	hash, found := tp.sources[source]
	if !found {
		return Transaction{}, false
	}
	return tp.Get(hash)
}

func (tp *Pool) Add(tx Transaction) error {
	txHash := tx.GetHash()
	if tp.Has(txHash) {
		return errors.TransactionAlreadyExistsInPool
	}

	metrics.TxPool.AddSize(1)

	tp.Lock()
	defer tp.Unlock()

	if len(tp.Pool) >= tp.limit {
		return errors.TransactionPoolFull
	}

	tp.Pool[txHash] = tx
	tp.sources[tx.Source()] = txHash

	e := tp.hashList.PushBack(txHash)
	tp.hashMap[txHash] = e

	return nil
}

func (tp *Pool) Remove(hashes ...string) {
	if len(hashes) < 1 {
		return
	}

	tp.Lock()
	defer tp.Unlock()

	var num int
	for _, hash := range hashes {
		if tx, found := tp.Pool[hash]; found {
			delete(tp.sources, tx.Source())
			delete(tp.Pool, hash)
			if e, ok := tp.hashMap[hash]; ok {
				tp.hashList.Remove(e)
				delete(tp.hashMap, hash)
			}
			num++
		}
	}

	metrics.TxPool.AddSize(-num)

}

func (tp *Pool) RemoveFromSources(sources ...string) {
	if len(sources) < 1 {
		return
	}

	tp.Lock()
	defer tp.Unlock()

	var num int
	for _, source := range sources {
		if hash, found := tp.sources[source]; found {
			if _, found := tp.Pool[hash]; found {
				delete(tp.sources, source)
				delete(tp.Pool, hash)
				if e, ok := tp.hashMap[hash]; ok {
					tp.hashList.Remove(e)
					delete(tp.hashMap, hash)
				}
				num++
			}
		}
	}

	metrics.TxPool.AddSize(-num)
}

func (tp *Pool) AvailableTransactions(transactionLimit int) []string {
	if transactionLimit < 1 {
		return nil
	}

	tp.RLock()
	defer tp.RUnlock()

	var ret []string
	var cnt int
	// first ouput by order older hash
	for e := tp.hashList.Front(); e != nil; e = e.Next() {
		if cnt >= transactionLimit {
			return ret
		}
		hash, ok := e.Value.(string)
		if ok {
			ret = append(ret, hash)
			cnt++
		}
	}

	return ret
}

func (tp *Pool) IsSameSource(source string) (found bool) {
	tp.RLock()
	defer tp.RUnlock()

	_, found = tp.sources[source]

	return
}
