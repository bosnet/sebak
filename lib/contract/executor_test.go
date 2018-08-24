package contract

import (
	"testing"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/contract/native/execfunc"
	"boscoin.io/sebak/lib/contract/payload"
	"boscoin.io/sebak/lib/contract/value"
	sebakstorage "boscoin.io/sebak/lib/storage"
)

func TestContractExecutorNativeHelloworld(t *testing.T) {
	st, err := sebakstorage.NewTestMemoryLevelDBBackend()
	if err != nil {
		t.Fatal(err)
	}

	ts, err := st.OpenTransaction()
	if err != nil {
		t.Fatal(err)
	}

	sdb := sebakstorage.NewStateDB(ts)

	senderAccount := block.TestMakeBlockAccount()
	ctx := NewContext(senderAccount, sdb)
	exCode := &payload.ExecCode{
		ContractAddress: execfunc.HelloWorldAddress,
		Method:          "hello",
	}

	ex, err := NewExecutor(ctx, exCode)
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
	if retCode.EqualNative("world") == false {
		t.Fatalf("retcode.Contents have:%v want:%v", retCode, "world")
	}
}
