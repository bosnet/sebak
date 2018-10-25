package resource

import (
	"strings"

	"boscoin.io/sebak/lib/block"
	"github.com/nvellon/hal"
)

type TransactionHistory struct {
	bt *block.BlockTransactionHistory
}

func NewTransactionHistory(bt *block.BlockTransactionHistory) *TransactionHistory {
	t := &TransactionHistory{
		bt: bt,
	}
	return t
}

func (t TransactionHistory) GetMap() hal.Entry {
	return hal.Entry{
		"hash":      t.bt.Hash,
		"source":    t.bt.Source,
		"confirmed": t.bt.Time,
		"created":   t.bt.Created,
		"status":    t.bt.Status,
		"message":   t.bt.Message,
	}
}
func (t TransactionHistory) Resource() *hal.Resource {
	r := hal.NewResource(t, t.LinkSelf())
	r.AddLink("account", hal.NewLink(strings.Replace(URLAccounts, "{id}", t.bt.Source, -1)))
	r.AddLink("transaction", hal.NewLink(strings.Replace(URLTransactionByHash, "{id}", t.bt.Hash, -1)))
	return r
}

func (t TransactionHistory) LinkSelf() string {
	return strings.Replace(URLTransactions, "{id}", t.bt.Hash, -1)
}
