package jsvm

import (
	"testing"

	"github.com/magiconair/properties/assert"

	"boscoin.io/sebak/lib/contract/api"
	"boscoin.io/sebak/lib/contract/context"
	"boscoin.io/sebak/lib/contract/payload"
)

func TestExecutor(t *testing.T) {
	deployCode := &payload.DeployCode{
		ContractAddress: testAddress,
		Code:            []byte(testCode),
		Type:            payload.JavaScript,
	}
	testStateClone.PutDeployCode(deployCode)

	context := &context.Context{
		StateStore: testStateStore,
	}
	api := api.NewAPI(context, testAddress, nil)
	ex := NewOttoExecutor(context, api, deployCode)
	excode := &payload.ExecCode{
		ContractAddress: testAddress,
		Method:          "Hello",
		Args:            []string{"boscoin"},
	}
	ret, _ := ex.Execute(excode)
	want, _ := api.Helloworld("boscoin")
	assert.Equal(t, string(ret.Contents), want)
	t.Log(string(ret.Contents))
}
