package sebak

import (
	"testing"

	"boscoin.io/sebak/lib/contract/native/execfunc"
	"boscoin.io/sebak/lib/contract/payload"
	"boscoin.io/sebak/lib/contract/value"
	sebakstorage "boscoin.io/sebak/lib/storage"
)

func Test_ContractExecutor_NativeHelloworld(t *testing.T) {
	st, err := sebakstorage.NewTestMemoryLevelDBBackend()
	if err != nil {
		t.Fatal(err)
	}
	stateStore := NewStateStore(st)
	stateClone := NewStateClone(stateStore)

	senderAccount := testMakeBlockAccount()
	ctx := &ContractContext{
		SenderAccount: senderAccount,
		StateStore:    stateStore,
		StateClone:    stateClone,
	}

	exCode := &payload.ExecCode{
		ContractAddress: execfunc.HelloWorldAddress,
		Method:          "hello",
	}

	ex, err := NewContractExecutor(ctx, exCode)
	if err != nil {
		t.Fatal(err)
	}

	retCode, err := ex.Execute(exCode)
	if err != nil {
		t.Fatal(err)
	}

	if retCode == nil {
		t.Fatal("retCode is nil")
	}

	if retCode.Type != value.String {
		t.Fatalf("retCode.Type have:%v want:%v", retCode.Type, value.String)
	}
	if string(retCode.Contents) != "world" {
		t.Fatalf("retcode.Contents have:%v want:%v", retCode.Contents, "world")
	}
}
