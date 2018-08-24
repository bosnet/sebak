package contract

import (
	"boscoin.io/sebak/lib/contract/context"
	"boscoin.io/sebak/lib/contract/jsvm"
	"boscoin.io/sebak/lib/contract/payload"
)

type Deployer interface {
	Deploy(code []byte) error
}

func NewDeployer(ctx *context.Context, codeType payload.CodeType) Deployer {

	var cd Deployer

	switch codeType {
	case payload.JavaScript:
		cd = jsvm.NewDeployer(ctx)
	default:
		panic("not yet supported")
	}
	return cd

}

func Deploy(ctx *context.Context, codeType payload.CodeType, code []byte) (err error) {
	cd := NewDeployer(ctx, codeType)
	err = cd.Deploy(code)
	return
}
