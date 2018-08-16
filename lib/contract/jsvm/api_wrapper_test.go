package jsvm

import (
	"testing"

	"boscoin.io/sebak/lib/contract/context"
	"boscoin.io/sebak/lib/contract/payload"
	"boscoin.io/sebak/lib/contract/test"
	"boscoin.io/sebak/lib/contract/value"
)

func TestJSVMCallContract(t *testing.T) {
	testAddress := "testadress"

	ctx := test.NewMockContext(testAddress, nil)
	api := test.NewMockAPI(ctx, testAddress)

	callc := func(ctx context.Context, execCode *payload.ExecCode) (*value.Value, error) {
		if execCode.ContractAddress != "helloworld" {
			t.Fatalf("execCode.ContractAddress have:%v want:%v", execCode.ContractAddress, "helloworld")
		}

		ret := &value.Value{
			Type:     value.String,
			Contents: []byte("world!"),
		}
		return ret, nil
	}

	api.SetCallContractFunc(callc)

	code := `
function HelloContract() {
	ret = CallContract("helloworld","hello","world")
	return ret
}
`
	deployCode := &payload.DeployCode{
		ContractAddress: testAddress,
		Code:            []byte(code),
		Type:            payload.JavaScript,
	}
	ex := NewOttoExecutor(ctx, api, deployCode)
	excode := &payload.ExecCode{
		ContractAddress: testAddress,
		Method:          "HelloContract",
	}
	ret, err := ex.Execute(excode)
	if err != nil {
		t.Fatal(err)
	}
	if ret == nil {
		t.Fatal("ret is nil")
	}
}
