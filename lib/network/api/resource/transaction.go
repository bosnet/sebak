package resource

import (
	"strings"

	"github.com/nvellon/hal"
)

type Transaction struct {
	hash       string
	sequenceID uint64
	signature  string
	source     string
	fee        string
	amount     string
	created    string
	operations []string
}

func (t Transaction) GetMap() hal.Entry {
	return hal.Entry{
		"id":              t.hash,
		"hash":            t.hash,
		"account":         t.source,
		"fee_paid":        t.fee,
		"sequence_id":     t.sequenceID,
		"created_at":      t.created,
		"operation_count": len(t.operations),
	}
}
func (t Transaction) Resource() *hal.Resource {

	r := hal.NewResource(t, t.LinkSelf())
	r.AddLink("accounts", hal.NewLink(strings.Replace(URLAccounts, "{id}", t.source, -1)))
	r.AddLink("operations", hal.NewLink(strings.Replace(URLTransactions, "{id}", t.hash, -1)+"/operations{?cursor,limit,order}", hal.LinkAttr{"templated": true}))
	return r
}

func (t Transaction) LinkSelf() string {
	return strings.Replace(URLTransactions, "{id}", t.hash, -1)
}
