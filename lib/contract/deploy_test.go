package contract

import (
	"testing"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/contract/payload"
	sebakstorage "boscoin.io/sebak/lib/storage"
	"github.com/stretchr/testify/assert"
)

func TestContractJSVMDeploy(t *testing.T) {
	testAddress := "testaddress"
	testCode := `
function Hello(helloarg){
    return HelloWorld(helloarg)
}
`

	st, err := sebakstorage.NewTestMemoryLevelDBBackend()
	if err != nil {
		t.Fatal(err)
	}

	ba := block.NewBlockAccount(testAddress, 1000000000000, "")
	if err := ba.Save(st); err != nil {
		t.Fatal(err)
	}

	ts, err := st.OpenTransaction()
	if err != nil {
		t.Fatal(err)
	}
	sdb := sebakstorage.NewStateDB(ts)
	ctx := NewContractContext(ba, sdb)

	if err := DeployContract(ctx, payload.JavaScript, []byte(testCode)); err != nil {
		t.Fatal(err)
	}

	deployCode, err := payload.GetDeployCode(sdb, testAddress)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, deployCode.Code, []byte(testCode))
	assert.Equal(t, deployCode.ContractAddress, testAddress)
	assert.Equal(t, deployCode.Type, payload.JavaScript)
}
