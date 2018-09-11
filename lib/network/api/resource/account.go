package resource

import (
	"strings"

	"github.com/nvellon/hal"
)

type Account struct {
	accountID  string
	sequenceID uint64
	balance    string
}

func (a Account) GetMap() hal.Entry {
	return hal.Entry{
		"id":          a.accountID,
		"account_id":  a.accountID,
		"sequence_id": a.sequenceID,
		"balance":     a.balance,
	}
}

func (a Account) Resource() *hal.Resource {
	r := hal.NewResource(a, a.LinkSelf())
	r.AddLink("transactions", hal.NewLink(strings.Replace(URLAccounts, "{id}", a.accountID, -1)+"/transactions{?cursor,limit,order}", hal.LinkAttr{"templated": true}))
	r.AddLink("operations", hal.NewLink(strings.Replace(URLAccounts, "{id}", a.accountID, -1)+"/operations{?cursor,limit,order}", hal.LinkAttr{"templated": true}))
	return r
}

func (a Account) LinkSelf() string {
	return strings.Replace(URLAccounts, "{id}", a.accountID, -1)
}
