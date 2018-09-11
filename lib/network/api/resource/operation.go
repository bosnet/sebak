package resource

import (
	"strings"

	"github.com/nvellon/hal"
)

type Operation struct {
	hash    string
	txHash  string
	funder  string //Source Account
	account string //Target Account
	otype   string
	amount  string
}

func (o Operation) GetMap() hal.Entry {
	return hal.Entry{
		"id":      o.hash,
		"hash":    o.hash,
		"funder":  o.funder,
		"account": o.account,
		"type":    o.otype,
		"amount":  o.amount,
	}
}

func (o Operation) Resource() *hal.Resource {

	r := hal.NewResource(o, o.LinkSelf())
	r.AddNewLink("transactions", strings.Replace(URLTransactions, "{id}", o.txHash, -1))
	return r
}

func (o Operation) LinkSelf() string {
	return strings.Replace(URLOperations, "{id}", o.hash, -1)
}
