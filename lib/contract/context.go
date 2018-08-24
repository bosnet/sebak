package contract

import (
	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/contract/payload"
	sebakstorage "boscoin.io/sebak/lib/storage"
)

type Context struct {
	senderAccount *block.BlockAccount // Transaction.Source.
	db            sebakstorage.DBBackend

	//APICallCounter int // Simple version of PC
}

func NewContext(sender *block.BlockAccount, db sebakstorage.DBBackend) *Context {
	ctx := &Context{
		senderAccount: sender,
		db:            db,
	}
	return ctx
}

func (c *Context) SenderAddress() string {
	return c.senderAccount.Address
}

func (c *Context) PutDeployCode(code *payload.DeployCode) error {
	return code.Save(c.db)
}
func (c *Context) GetDeployCode(address string) (*payload.DeployCode, error) {
	return payload.GetDeployCode(c.db, address)
}
