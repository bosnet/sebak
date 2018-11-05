package operation

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/common/keypair"
)

func TestMakeHashOfOperationBodyPayment(t *testing.T) {
	kp := keypair.Master("find me")

	opb := Payment{
		Target: kp.Address(),
		Amount: common.Amount(100),
	}
	op := Operation{
		H: Header{Type: TypePayment},
		B: opb,
	}
	hashed := op.MakeHashString()

	expected := "24V5mcAAoUX1oSn7pqUgZPGN7MxWVtRxZQ9Pc3yn1SmD"
	require.Equal(t, hashed, expected)
}

func TestIsWellFormedOperation(t *testing.T) {
	op := TestMakeOperation(-1)
	err := op.IsWellFormed(networkID, common.NewConfig())
	require.NoError(t, err)
}

func TestIsWellFormedOperationLowerAmount(t *testing.T) {
	obp := TestMakeOperationBodyPayment(0)
	err := obp.IsWellFormed(networkID, common.NewConfig())
	require.Error(t, err)
}

func TestSerializeOperation(t *testing.T) {
	op := TestMakeOperation(-1)
	b, err := op.Serialize()
	require.NoError(t, err)
	require.Equal(t, len(b) > 0, true)

	var o Operation
	err = json.Unmarshal(b, &o)
	require.NoError(t, err)
}

func TestOperationBodyCongressVoting(t *testing.T) {
	opb := NewCongressVoting([]byte("dummy contract"), 1, 100)
	op := Operation{
		H: Header{Type: TypeCongressVoting},
		B: opb,
	}
	hashed := op.MakeHashString()

	expected := "4CcZvkNYQUgvdmjGDuMx7tesCdRp3HU4CW3pbRxeqtEZ"
	require.Equal(t, hashed, expected)

	err := op.IsWellFormed(networkID, common.NewConfig())
	require.NoError(t, err)

}

func TestOperationBodyCongressVotingResult(t *testing.T) {
	opb := NewCongressVotingResult(
		string(common.MakeHash([]byte("dummydummy"))),
		[]string{"http://www.boscoin.io/1", "http://www.boscoin.io/2"},
		string(common.MakeHash([]byte("dummydummy"))),
		[]string{"http://www.boscoin.io/3", "http://www.boscoin.io/4"},
		9, 2, 3, 4,
	)
	op := Operation{
		H: Header{Type: TypeCongressVotingResult},
		B: opb,
	}
	hashed := op.MakeHashString()

	expected := "8DgD3heMuNLYhnNBgPSBEquAdKXuogrSybdqt7WD87CV"
	require.Equal(t, hashed, expected)

	err := op.IsWellFormed(networkID, common.NewConfig())
	require.NoError(t, err)

}
