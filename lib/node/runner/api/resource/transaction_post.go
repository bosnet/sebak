package resource

import (
	"strings"

	"boscoin.io/sebak/lib/transaction"
	"github.com/nvellon/hal"
)

type TransactionPost struct {
	tx   transaction.Transaction
	hash string
}

func NewTransactionPost(tx transaction.Transaction) *TransactionPost {
	t := &TransactionPost{
		tx:   tx,
		hash: tx.B.MakeHashString(),
	}
	return t
}

func (t TransactionPost) GetMap() hal.Entry {
	return hal.Entry{
		"hash":    t.hash,
		"status":  "submitted",
		"message": t.tx.B,
	}
}
func (t TransactionPost) Resource() *hal.Resource {
	r := hal.NewResource(t, t.LinkSelf())
	//r.AddLink("history", hal.NewLink(strings.Replace(URLTransactionHistory, "{id}", t.hash, -1)))
	return r
}

func (t TransactionPost) LinkSelf() string {
	return strings.Replace(URLTransactions, "{id}", t.tx.H.Hash, -1)
}
