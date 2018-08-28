package contract

import (
	"testing"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/contract/context"
	"boscoin.io/sebak/lib/contract/payload"
	"boscoin.io/sebak/lib/contract/value"
	"boscoin.io/sebak/lib/statedb"
	sebakstorage "boscoin.io/sebak/lib/storage"
	"boscoin.io/sebak/lib/trie"
)

func TestContractExecutorNativeHelloworld(t *testing.T) {
	testAddress := "testaddress"
	testCode := `
function Hello(helloarg){
    return HelloWorld(helloarg)
}
`
	st, _ := sebakstorage.NewTestMemoryLevelDBBackend()
	Root := sebakcommon.Hash{}

	// Deploy hello world js
	{
		sdb := statedb.New(Root, trie.NewEthDatabase(st))

		sdb.CreateAccount(testAddress)

		ctx := context.NewContext(testAddress, sdb)

		if err := Deploy(ctx, payload.JavaScript, []byte(testCode)); err != nil {
			t.Fatal(err)
		}
		Root, _ = sdb.CommitTrie()
		sdb.CommitDB(Root)
	}

	// Execute hello world js
	{
		sdb := statedb.New(Root, trie.NewEthDatabase(st))
		ctx := context.NewContext(testAddress, sdb)

		exCode := &payload.ExecCode{
			ContractAddress: testAddress,
			Method:          "Hello",
			Args:            []interface{}{"boscoin"},
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
		if retCode.EqualNative("boscoin WORLD!!") == false {
			t.Fatalf("retcode.Contents have:%v want:%v", retCode, "boscoin WORLD!!")
		}
	}

	// Execute hello world Native function
	{

		sdb := statedb.New(sebakcommon.Hash{}, trie.NewEthDatabase(st))
		ctx := context.NewContext(testAddress, sdb)
		exCode := &payload.ExecCode{
			ContractAddress: "HELLOWORLDADDRESS",
			Method:          "Hello",
			Args:            []interface{}{"boscoin"},
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
		if retCode.EqualNative("boscoin WORLD!!") == false {
			t.Fatalf("retcode.Contents have:%v want:%v", retCode, "boscoin WORLD!!")
		}
	}
}
