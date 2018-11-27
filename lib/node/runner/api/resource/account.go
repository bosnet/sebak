package resource

import (
	"boscoin.io/sebak/lib/common"
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
		"address":     a.ba.Address,
		"sequence_id": a.ba.SequenceID,
		"balance":     a.ba.Balance,
		"linked":      a.ba.Linked,
	}
}

func (a Account) Resource() *hal.Resource {
	address := a.ba.Address
	accountID := a.ba.Address

	r := hal.NewResource(a, a.LinkSelf())
	r.AddLink("transactions", hal.NewLink(strings.Replace(URLAccountTransactions, "{id}", address, -1)+"{?cursor,limit,order}", hal.LinkAttr{"templated": true}))
	r.AddLink("operations", hal.NewLink(strings.Replace(URLAccountOperations, "{id}", accountID, -1)+"{?cursor,limit,order}", hal.LinkAttr{"templated": true}))
	return r
}

func (a Account) LinkSelf() string {
	address := a.ba.Address
	return strings.Replace(URLAccounts, "{id}", address, -1)
}

func (a Account) MarshalJSON() ([]byte, error) {
	r := a.Resource()
	return common.JSONMarshalWithoutEscapeHTML(r.GetMap())
}
