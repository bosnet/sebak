package jsvm

import (
	"testing"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/contract/api"
	"boscoin.io/sebak/lib/contract/context"
	"boscoin.io/sebak/lib/contract/payload"
	"boscoin.io/sebak/lib/contract/test"
	"boscoin.io/sebak/lib/contract/value"
)

func newTestOttoExecutor(ctx context.Context, api api.API, addr, code string) *OttoExecutor {
	deployCode := &payload.DeployCode{
		ContractAddress: addr,
		Code:            []byte(code),
		Type:            payload.JavaScript,
	}
	ex := NewOttoExecutor(ctx, api, deployCode)
	return ex
}

func TestJSVMGetBalance(t *testing.T) {
	a := "testaddress"
	ctx := test.NewMockContext(a, nil)
	api := test.NewMockAPI(ctx, a)
	getBalF := func(ctx context.Context) (sebakcommon.Amount, error) {
		amt := sebakcommon.Amount(100)
		return amt, nil
	}
	api.SetGetBalanceFunc(getBalF)

	code := `
function Hello() {
	ret = GetBalance()
	return ret
}
`
	ex := newTestOttoExecutor(ctx, api, a, code)

	excode := &payload.ExecCode{
		ContractAddress: a,
		Method:          "Hello",
	}
	ret, err := ex.Execute(excode)
	if err != nil {
		t.Fatal(err)
	}
	if ret == nil {
		t.Fatal("ret is nil")
	}

	if ret.Type != value.Int {
		t.Errorf("ret.Type have:%v want:%v", ret.Type, value.Int)
	}
}

func TestJSVMCallContract(t *testing.T) {
	testAddress := "testaddress"

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
	ex := newTestOttoExecutor(ctx, api, testAddress, code)
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
