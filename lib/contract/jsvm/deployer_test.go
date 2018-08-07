package jsvm

import (
	"testing"

	"boscoin.io/sebak/lib"
	"github.com/magiconair/properties/assert"

	"boscoin.io/sebak/lib/contract/context"
	"boscoin.io/sebak/lib/contract/payload"
)

func TestDeployer(t *testing.T) {

	ba := sebak.NewBlockAccount(testAddress, 1000000000000, "")
	ba.Save(testLevelDBBackend)

	context := &context.Context{
		SenderAccount: ba,
		StateStore:    testStateStore,
		StateClone:    testStateClone,
	}
	deployer := NewDeployer(context)

	deployer.Deploy([]byte(testCode))

	deployCode, err := testStateClone.GetDeployCode(testAddress)
	if err != nil {
		panic(err)
	}

	assert.Equal(t, deployCode.Code, []byte(testCode))
	assert.Equal(t, deployCode.ContractAddress, testAddress)
	assert.Equal(t, deployCode.Type, payload.JavaScript)
}
