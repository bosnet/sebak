package resource

import (
	"strings"

	"boscoin.io/sebak/lib/transaction"
	"github.com/nvellon/hal"
)

type TransactionPost struct {
	tx transaction.Transaction
}

func NewTransactionPost(tx transaction.Transaction) *TransactionPost {
	t := &TransactionPost{
		tx: tx,
	}
	return t
}

func (t TransactionPost) GetMap() hal.Entry {
	return hal.Entry{
		"hash":   t.tx.H.Hash,
		"status": "blah",
	}
}
func (t TransactionPost) Resource() *hal.Resource {
	r := hal.NewResource(t, t.LinkSelf())
	return r
}

func (t TransactionPost) LinkSelf() string {
	return strings.Replace(URLTransactions, "{id}", t.tx.H.Hash, -1)
}
