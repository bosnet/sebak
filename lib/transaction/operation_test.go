package transaction

import (
	"testing"

	"boscoin.io/sebak/lib/common"

	"encoding/json"
	"github.com/stellar/go/keypair"
	"github.com/stretchr/testify/require"
)

func TestMakeHashOfOperationBodyPayment(t *testing.T) {
	kp := keypair.Master("find me")

	opb := OperationBodyPayment{
		Target: kp.Address(),
		Amount: common.Amount(100),
	}
	op := Operation{
		H: OperationHeader{Type: OperationPayment},
		B: opb,
	}
	hashed := op.MakeHashString()

	expected := "8AALKhfgCu2w3ZtbESXHG5ko93Jb1L1yCmFopoJubQh9"
	require.Equal(t, hashed, expected)
}

func TestIsWellFormedOperation(t *testing.T) {
	op := TestMakeOperation(-1)
	err := op.IsWellFormed(networkID)
	require.Nil(t, err)
}

func TestIsWellFormedOperationLowerAmount(t *testing.T) {
	obp := TestMakeOperationBodyPayment(0)
	err := obp.IsWellFormed(networkID)
	require.NotNil(t, err)
}

func TestSerializeOperation(t *testing.T) {
	op := TestMakeOperation(-1)
	b, err := op.Serialize()
	require.Nil(t, err)
	require.Equal(t, len(b) > 0, true)

	var o Operation
	err = json.Unmarshal(b, &o)
	require.Nil(t, err)
}
