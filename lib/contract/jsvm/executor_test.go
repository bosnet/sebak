package jsvm

import (
	"fmt"
	"testing"

	"github.com/magiconair/properties/assert"

	"boscoin.io/sebak/lib/contract/context"
	"boscoin.io/sebak/lib/contract/payload"
	"boscoin.io/sebak/lib/contract/api"
)

func TestExecutor(t *testing.T) {
	deployCode := &payload.DeployCode{
		ContractAddress: testAddress,
		Code: []byte(testCode),
		Type: payload.JavaScript,
	}
	testStateClone.PutDeployCode(deployCode)

	context := &context.Context{
		StateStore: testStateStore,
	}
	ex := NewOttoExecutor(context, deployCode)
	excode := &payload.ExecCode{
		ContractAddress: testAddress,
		Method:          "Hello",
		Args:            []string{"boscoin"},
	}
	ret, _ := ex.Execute(excode)
	want, _ := api.NewAPI(context).Helloworld("boscoin")
	assert.Equal(t, string(ret.Contents), want)
	fmt.Println(string(ret.Contents))
}
