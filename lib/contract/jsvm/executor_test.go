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
	testHelloWorldAddress := "helloWorldAddr"
	testHelloWorldJS := `
function Hello(helloarg){
    return HelloWorld(helloarg)
}
`
	st, _ := sebakstorage.NewTestMemoryLevelDBBackend()
	var Root = sebakcommon.Hash{}

	//Deploy hello world js
	{
		sdb := statedb.New(Root, trie.NewEthDatabase(st))

		deployCode := &payload.DeployCode{
			ContractAddress: testHelloWorldAddress,
			Code:            []byte(testHelloWorldJS),
			Type:            payload.JavaScript,
		}

		ctx := context.NewContext(testHelloWorldAddress, sdb)
		deployer := NewDeployer(ctx)
		deployer.Deploy(deployCode.Code)
		Root, _ = sdb.CommitTrie()
		sdb.CommitDB(Root)
	}

	//Execute hello world js
	{
		sdb := statedb.New(Root, trie.NewEthDatabase(st))
		ctx := context.NewContext(testHelloWorldAddress, sdb)
		deployCode, err := ctx.GetDeployCode(testHelloWorldAddress)
		if err != nil {
			t.Error(err)
		}
		api := api.NewAPI(ctx, testHelloWorldAddress)

		ex := NewOttoExecutor(ctx, api, deployCode)
		excode := &payload.ExecCode{
			ContractAddress: testHelloWorldAddress,
			Method:          "Hello",
			Args:            []interface{}{"boscoin"},
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

	testGetSetAddress := "getSetAddr"
	testGetSetJS := `
function Set(arg){
    return SetStatus("arg", arg)
}

function Get(arg){
    return GetStatus("arg")
}
`

	//Deploy get set status js
	{
		sdb := statedb.New(Root, trie.NewEthDatabase(st))

		deployCode := &payload.DeployCode{
			ContractAddress: testGetSetAddress,
			Code:            []byte(testGetSetJS),
			Type:            payload.JavaScript,
		}

		ctx := context.NewContext(testGetSetAddress, sdb)
		deployer := NewDeployer(ctx)
		deployer.Deploy(deployCode.Code)
		Root, _ = sdb.CommitTrie()
		sdb.CommitDB(Root)
	}

	//Execute Set function
	{
		sdb := statedb.New(Root, trie.NewEthDatabase(st))
		ctx := context.NewContext(testGetSetAddress, sdb)
		deployCode, err := ctx.GetDeployCode(testGetSetAddress)
		if err != nil {
			t.Error(err)
		}
		api := api.NewAPI(ctx, testGetSetAddress)

		ex := NewOttoExecutor(ctx, api, deployCode)
		excode := &payload.ExecCode{
			ContractAddress: testGetSetAddress,
			Method:          "Set",
			Args:            []interface{}{"boscoin"},
		}

		ret, err := ex.Execute(excode)
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, true, ret.Value())

		Root, _ = sdb.CommitTrie()
		sdb.CommitDB(Root)
	}

	//Execute Get function
	{
		sdb := statedb.New(Root, trie.NewEthDatabase(st))
		ctx := context.NewContext(testGetSetAddress, sdb)
		deployCode, err := ctx.GetDeployCode(testGetSetAddress)
		if err != nil {
			t.Error(err)
		}
		api := api.NewAPI(ctx, testGetSetAddress)

		ex := NewOttoExecutor(ctx, api, deployCode)
		excode := &payload.ExecCode{
			ContractAddress: testGetSetAddress,
			Method:          "Get",
			Args:            []interface{}{"boscoin"},
		}

		ret, err := ex.Execute(excode)
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, "boscoin", ret.Value())
	}
}
