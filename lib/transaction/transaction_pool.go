package transaction

import (
	"boscoin.io/sebak/lib/common"
)

type TransactionPool struct {
	common.SafeLock

	Pool    map[ /* Transaction.GetHash() */ string]Transaction
	Hashes  []string // Transaction.GetHash()
	Sources map[ /* Transaction.Source() */ string]bool
}

func NewTransactionPool() *TransactionPool {
	return &TransactionPool{
		Pool:    map[string]Transaction{},
		Hashes:  []string{},
		Sources: map[string]bool{},
	}
}

func (tp *TransactionPool) Len() int {
	return len(tp.Hashes)
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
	if _, found := tp.Pool[tx.GetHash()]; found {
		return false
	}

	tp.Lock()
	defer tp.Unlock()

	tp.Pool[tx.GetHash()] = tx
	tp.Hashes = append(tp.Hashes, tx.GetHash())
	tp.Sources[tx.Source()] = true

	return true
}

func (tp *TransactionPool) Remove(hashes ...string) {
	if len(hashes) < 1 {
		return
	}

	tp.Lock()
	defer tp.Unlock()

	indices := map[int]int{}
	var max int
	for _, hash := range hashes {
		index, found := common.InStringArray(tp.Hashes, hash)
		if !found {
			continue
		}
		indices[index] = 1
		if index > max {
			max = index
		}

		if tx, found := tp.Get(hash); found {
			delete(tp.Sources, tx.Source())
		}
	}

	var newHashes []string
	for i, hash := range tp.Hashes {
		if i > max {
			newHashes = append(newHashes, hash)
			continue
		}

		if _, found := indices[i]; !found {
			newHashes = append(newHashes, hash)
			continue
		}

		delete(tp.Pool, hash)
	}

	tp.Hashes = newHashes

	return
}

func (tp *TransactionPool) AvailableTransactions(transactionLimit uint64) []string {
	tp.Lock()
	defer tp.Unlock()

	if tp.Len() <= int(transactionLimit) {
		return tp.Hashes
	}

	return tp.Hashes[:transactionLimit]
}

func (tp *TransactionPool) IsSameSource(source string) (found bool) {
	_, found = tp.Sources[source]

	return
}
