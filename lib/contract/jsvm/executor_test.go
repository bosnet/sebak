package jsvm

import (
	"testing"

	"boscoin.io/sebak/lib/contract/context"
	"boscoin.io/sebak/lib/contract/payload"
	"boscoin.io/sebak/lib/contract/test"
	"github.com/stretchr/testify/assert"
)

func TestJSVMExecutor(t *testing.T) {
	testAddress := "testadress"
	testCode := `
function Hello(helloarg){
    return HelloWorld(helloarg)
}
`

	deployCode := &payload.DeployCode{
		ContractAddress: testAddress,
		Code:            []byte(testCode),
		Type:            payload.JavaScript,
	}

	ctx := test.NewMockContext(testAddress, nil)
	api := test.NewMockAPI(ctx, testAddress)
	api.SetHelloworldFunc(func(ctx context.Context, greeting string) (string, error) {
		return greeting + " WORLD!!", nil
	})

	ex := NewOttoExecutor(ctx, api, deployCode)
	excode := &payload.ExecCode{
		ContractAddress: testAddress,
		Method:          "Hello",
		Args:            []string{"boscoin"},
	}

	ret, err := ex.Execute(excode)
	if err != nil {
		t.Fatal(err)
	}
	want, err := api.Helloworld("boscoin")
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, string(ret.Contents), want)
	t.Log(string(ret.Contents))
}
