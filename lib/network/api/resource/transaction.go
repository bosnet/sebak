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
		"id":              t.bt.Hash,
		"hash":            t.bt.Hash,
		"account":         t.bt.Source,
		"fee_paid":        t.bt.Fee.String(),
		"sequence_id":     t.bt.SequenceID,
		"created_at":      t.bt.Created,
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
