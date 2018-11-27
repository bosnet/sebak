package resource

import (
	"boscoin.io/sebak/lib/common"
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
	return r
}

func (t TransactionPost) LinkSelf() string {
	return strings.Replace(URLTransactions, "{id}", t.tx.H.Hash, -1)
}

func (t TransactionPost) MarshalJSON() ([]byte, error) {
	r := t.Resource()
	return common.JSONMarshalWithoutEscapeHTML(r.GetMap())
}
