package sebak

import (
	"boscoin.io/sebak/lib/contract/jsvm"
	"boscoin.io/sebak/lib/contract/payload"
)

type Deployer interface {
	Deploy(code []byte) error
}

func NewDeployer(ctx *ContractContext, codeType payload.CodeType) Deployer {

	var de Deployer

	switch codeType {
	case payload.JavaScript:
		de = jsvm.NewDeployer(ctx)
	default:
		panic("not yet supported")
	}
	return de

}

func Deploy(ctx *ContractContext, codeType payload.CodeType, code []byte) (err error) {
	deployer := NewDeployer(ctx, codeType)
	err = deployer.Deploy(code)
	return
}
