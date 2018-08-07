package contract

import (
	"testing"

	sebak "boscoin.io/sebak/lib"
	"boscoin.io/sebak/lib/contract/context"
	"boscoin.io/sebak/lib/contract/native/execfunc"
	"boscoin.io/sebak/lib/contract/payload"
	"boscoin.io/sebak/lib/contract/value"
)

func Test_Executor_Native_HelloWorld(t *testing.T) {
	ctx := &context.Context{
		SenderAccount: sebak.NewBlockAccount("sender1", sebak.Amount(1000), "tx1-tx1"),
		StateStore:    testStateStore,
		StateClone:    testStateClone,
	}

	exCode := &payload.ExecCode{
		ContractAddress: execfunc.HelloWorldAddress,
		Method:          "hello",
	}

	ex, err := NewExecutor(ctx, exCode)
	if err != nil {
		t.Error(err)
		return
	}

	retCode, err := ex.Execute(exCode)
	if err != nil {
		t.Error(err)
		return
	}

	{
		v := retCode
		if v.Type != value.String {
			t.Errorf("v.Type have:%v want:%v", v.Type, value.String)
			return
		}
		if string(v.Contents) != "world" {
			t.Errorf("v.Contents have:%s want:%v", v.Contents, "world")
			return
		}
	}

	if retCode == nil {
		t.Errorf("retCode have:%v want:%v", retCode, nil)
		return
	}
}
