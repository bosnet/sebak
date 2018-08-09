package sebak

import (
	"boscoin.io/sebak/lib/contract/jsvm"
	"boscoin.io/sebak/lib/contract/payload"
)

type ContractDeployer interface {
	Deploy(code []byte) error
}

func NewContractDeployer(ctx *ContractContext, codeType payload.CodeType) ContractDeployer {

	var cd ContractDeployer

	switch codeType {
	case payload.JavaScript:
		cd = jsvm.NewDeployer(ctx)
	default:
		panic("not yet supported")
	}
	return cd

}

func DeployContract(ctx *ContractContext, codeType payload.CodeType, code []byte) (err error) {
	cd := NewContractDeployer(ctx, codeType)
	err = cd.Deploy(code)
	return
}
