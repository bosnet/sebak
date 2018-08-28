package jsvm

import (
	"boscoin.io/sebak/lib/contract/context"
	"boscoin.io/sebak/lib/contract/payload"
	"github.com/robertkrimen/otto"
)

type JsDeployer struct {
	Context *context.Context
}

func NewDeployer(context *context.Context) *JsDeployer {

	de := &JsDeployer{
		Context: context,
	}
	return de
}

func (jd *JsDeployer) Deploy(codeByte []byte) (err error) {

	var deployCode = new(payload.DeployCode)
	deployCode.ContractAddress = jd.Context.SenderAddress()
	deployCode.Code = codeByte
	deployCode.Type = payload.JavaScript

	if err = jd.compile(deployCode.Code); err != nil {
		return
	}

	jd.Context.PutDeployCode(deployCode)
	return nil
}

func (jd *JsDeployer) compile(code []byte) (err error) {
	vm := otto.New()

	_, err = vm.Compile("", code)

	return
}
