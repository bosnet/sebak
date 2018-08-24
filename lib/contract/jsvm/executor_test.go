package jsvm

import (
	"testing"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/contract/api"
	"boscoin.io/sebak/lib/contract/context"
	"boscoin.io/sebak/lib/contract/payload"
	"boscoin.io/sebak/lib/statedb"
	"boscoin.io/sebak/lib/storage"
	"boscoin.io/sebak/lib/trie"
	"github.com/stretchr/testify/assert"
)

func Test_JSVM_Executor(t *testing.T) {
	testAddress := "testadress"
	testCode := `
function Hello(helloarg){
    return HelloWorld(helloarg)
}
`
	st, _ := sebakstorage.NewTestMemoryLevelDBBackend()
	var Root = sebakcommon.Hash{}
	{
		sdb := statedb.New(Root, trie.NewEthDatabase(st))

		deployCode := &payload.DeployCode{
			ContractAddress: testAddress,
			Code:            []byte(testCode),
			Type:            payload.JavaScript,
		}

		ctx := context.NewContext(testAddress, sdb)
		deployer := NewDeployer(ctx)
		deployer.Deploy(deployCode.Code)
		Root, _ = sdb.CommitTrie()
		sdb.CommitDB(Root)
	}
	{
		sdb := statedb.New(Root, trie.NewEthDatabase(st))
		ctx := context.NewContext(testAddress, sdb)
		deployCode, err := ctx.GetDeployCode(testAddress)
		if err != nil {
			t.Error(err)
		}
		api := api.NewAPI(ctx, testAddress)

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

		assert.Equal(t, ret.String(), want)
	}

}
