package contract

import (
	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/contract/payload"
	sebakstorage "boscoin.io/sebak/lib/storage"
)

type ContractContext struct {
	senderAccount *block.BlockAccount // Transaction.Source.
	db            sebakstorage.DBBackend

	//APICallCounter int // Simple version of PC
}

func NewContractContext(sender *block.BlockAccount, db sebakstorage.DBBackend) *ContractContext {
	ctx := &ContractContext{
		senderAccount: sender,
		db:            db,
	}
	return ctx
}

func (c *ContractContext) SenderAddress() string {
	return c.senderAccount.Address
}

func (c *ContractContext) PutDeployCode(code *payload.DeployCode) error {
	return code.Save(c.db)
}
func (c *ContractContext) GetDeployCode(address string) (*payload.DeployCode, error) {
	return payload.GetDeployCode(c.db, address)
}
