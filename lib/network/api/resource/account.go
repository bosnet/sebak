package resource

import (
	"strings"

	"github.com/nvellon/hal"

	"boscoin.io/sebak/lib/block"
)

type Account struct {
	ba *block.BlockAccount
}

func NewAccount(ba *block.BlockAccount) *Account {
	a := &Account{
		ba: ba,
	}
	return a
}

func (a Account) GetMap() hal.Entry {
	return hal.Entry{
		"id":          a.ba.Address,
		"account_id":  a.ba.Address,
		"sequence_id": a.ba.SequenceID,
		"balance":     a.ba.Balance,
	}
}

func (a Account) Resource() *hal.Resource {
	address := a.ba.Address
	accountID := a.ba.Address

	r := hal.NewResource(a, a.LinkSelf())
	r.AddLink("transactions", hal.NewLink(strings.Replace(URLAccounts, "{id}", address, -1)+"/transactions{?cursor,limit,order}", hal.LinkAttr{"templated": true}))
	r.AddLink("operations", hal.NewLink(strings.Replace(URLAccounts, "{id}", accountID, -1)+"/operations{?cursor,limit,order}", hal.LinkAttr{"templated": true}))
	return r
}

func (a Account) LinkSelf() string {
	address := a.ba.Address
	return strings.Replace(URLAccounts, "{id}", address, -1)
}
