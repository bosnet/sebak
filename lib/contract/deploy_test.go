package contract

import (
	"testing"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/contract/context"
	"boscoin.io/sebak/lib/contract/payload"
	"boscoin.io/sebak/lib/statedb"
	sebakstorage "boscoin.io/sebak/lib/storage"
	"boscoin.io/sebak/lib/trie"
	"github.com/stretchr/testify/require"
)

func TestContractJSVMDeploy(t *testing.T) {
	testAddress := "testaddress"
	testCode := `
function Hello(helloarg){
    return HelloWorld(helloarg)
}
`
	st, _ := sebakstorage.NewTestMemoryLevelDBBackend()
	Root := sebakcommon.Hash{}
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
	{
		sdb := statedb.New(Root, trie.NewEthDatabase(st))
		ctx := context.NewContext(testAddress, sdb)
		deployCode, err := ctx.GetDeployCode(testAddress)
		if err != nil {
			t.Fatal(err)
		}

		require.Equal(t, deployCode.Code, []byte(testCode))
		require.Equal(t, deployCode.ContractAddress, testAddress)
		require.Equal(t, deployCode.Type, payload.JavaScript)
	}
}
