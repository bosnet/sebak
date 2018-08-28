package context

import (
	"boscoin.io/sebak/lib/contract/payload"
	"boscoin.io/sebak/lib/statedb"
)

type Context struct {
	stateDB *statedb.StateDB
	sender  string

	//APICallCounter int // Simple version of PC
}

func NewContext(senderAddr string, stateDB *statedb.StateDB) *Context {
	ctx := &Context{
		stateDB: stateDB,
		sender:  senderAddr,
	}
	return ctx
}

func (c *Context) SenderAddress() string {
	return c.sender
}

func (c *Context) PutDeployCode(code *payload.DeployCode) (err error) {
	encoded, err := code.Serialize()
	if err == nil {
		c.stateDB.SetCode(c.sender, encoded)
	}
	return
}
func (c *Context) GetDeployCode(address string) (deployCode *payload.DeployCode, err error) {
	encoded := c.stateDB.GetCode(c.sender)
	deployCode = &payload.DeployCode{}
	err = deployCode.Deserialize(encoded)
	return
}

func (c *Context) StateDB() *statedb.StateDB {
	return c.stateDB
}
