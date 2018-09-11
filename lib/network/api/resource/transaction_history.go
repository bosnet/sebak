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
		"id":        t.bt.Hash,
		"hash":      t.bt.Hash,
		"account":   t.bt.Source,
		"confirmed": t.bt.Confirmed,
		"created":   t.bt.Created,
		"message":   t.bt.Message,
	}
}
func (t TransactionHistory) Resource() *hal.Resource {

	r := hal.NewResource(t, t.LinkSelf())
	r.AddLink("accounts", hal.NewLink(strings.Replace(URLAccounts, "{id}", t.bt.Source, -1)))
	r.AddLink("operations", hal.NewLink(strings.Replace(URLTransactions, "{id}", t.bt.Hash, -1)+"/operations{?cursor,limit,order}", hal.LinkAttr{"templated": true}))
	return r
}

func (t TransactionHistory) LinkSelf() string {
	return strings.Replace(URLTransactions, "{id}", t.bt.Hash, -1)
}
