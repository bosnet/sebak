package sebak

import (
	"testing"

	"boscoin.io/sebak/lib/contract/payload"
	sebakstorage "boscoin.io/sebak/lib/storage"
	"github.com/stretchr/testify/assert"
)

func Test_Contract_JSVM_Deploy(t *testing.T) {
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

	store := NewStateStore(st)
	clone := NewStateClone(store)

	ba := NewBlockAccount(testAddress, 1000000000000, "")
	if err := ba.Save(st); err != nil {
		t.Fatal(err)
	}

	ctx := &ContractContext{
		SenderAccount: ba,
		StateStore:    store,
		StateClone:    clone,
	}

	if err := DeployContract(ctx, payload.JavaScript, []byte(testCode)); err != nil {
		t.Fatal(err)
	}

	deployCode, err := clone.GetDeployCode(testAddress)

	assert.Equal(t, deployCode.Code, []byte(testCode))
	assert.Equal(t, deployCode.ContractAddress, testAddress)
	assert.Equal(t, deployCode.Type, payload.JavaScript)

}
