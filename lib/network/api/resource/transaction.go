package resource

import (
	"strings"

	"boscoin.io/sebak/lib/block"
	"github.com/nvellon/hal"
)

type Transaction struct {
	bt *block.BlockTransaction
}

func NewTransaction(bt *block.BlockTransaction) *Transaction {
	t := &Transaction{
		bt: bt,
	}
	return t
}

func (t Transaction) GetMap() hal.Entry {
	return hal.Entry{
		"hash":            t.bt.Hash,
		"source":          t.bt.Source,
		"fee":             t.bt.Fee.String(),
		"sequenceid":     t.bt.SequenceID,
		"created":         t.bt.Created,
		"operation_count": len(t.bt.Operations),
	}
}
func (t Transaction) Resource() *hal.Resource {

	r := hal.NewResource(t, t.LinkSelf())
	r.AddLink("accounts", hal.NewLink(strings.Replace(URLAccounts, "{id}", t.bt.Source, -1)))
	r.AddLink("operations", hal.NewLink(strings.Replace(URLTransactions, "{id}", t.bt.Hash, -1)+"/operations{?cursor,limit,order}", hal.LinkAttr{"templated": true}))
	return r
}

func (t Transaction) LinkSelf() string {
	return strings.Replace(URLTransactions, "{id}", t.bt.Hash, -1)
}
