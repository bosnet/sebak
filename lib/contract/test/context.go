package test

import "boscoin.io/sebak/lib/contract/payload"

type DeployCodeFunc func(*payload.DeployCode) error

type MockContext struct {
	senderAddress  string
	deployCodeFunc DeployCodeFunc
}

func NewMockContext(addr string, dcFunc DeployCodeFunc) *MockContext {
	ctx := &MockContext{
		senderAddress:  addr,
		deployCodeFunc: dcFunc,
	}

	return ctx
}

func (ctx *MockContext) SenderAddress() string {
	return ctx.senderAddress
}

func (ctx *MockContext) PutDeployCode(code *payload.DeployCode) error {
	return ctx.deployCodeFunc(code)
}
