package sebak

import "boscoin.io/sebak/lib/contract/payload"

type ContractContext struct {
	SenderAccount *BlockAccount // Transaction.Source.
	//Transaction   *types.Transaction
	//BlockHeader   *types.BlockHeader

	StateStore *StateStore
	StateClone *StateClone

	//APICallCounter int // Simple version of PC
}

func (c *ContractContext) SenderAddress() string {
	return c.SenderAccount.Address
}

func (c *ContractContext) PutDeployCode(code *payload.DeployCode) error {
	return c.StateClone.PutDeployCode(code)
}
