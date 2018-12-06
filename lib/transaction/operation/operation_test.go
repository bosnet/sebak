package operation

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/common/keypair"
	"boscoin.io/sebak/lib/errors"
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

	expected := "GodiXQkWvAAbobhLBnWK8QS8aArb1ZoR2Ms8JswYUvL3"
	require.Equal(t, expected, hashed)
}

func TestIsWellFormedOperation(t *testing.T) {
	op := MakeTestPayment(-1)
	err := op.IsWellFormed(common.NewTestConfig())
	require.NoError(t, err)
}

func TestIsWellFormedOperationLowerAmount(t *testing.T) {
	kp := keypair.Random()
	obp := Payment{
		Target: kp.Address(),
		Amount: common.Amount(0),
	}
	err := obp.IsWellFormed(common.NewTestConfig())
	require.Error(t, err)
}

func TestSerializeOperation(t *testing.T) {
	op := MakeTestPayment(-1)
	b := common.MustMarshalJSON(op)
	require.Equal(t, len(b) > 0, true)

	var o Operation
	err := json.Unmarshal(b, &o)
	require.NoError(t, err)
}

func TestOperationBodyCongressVoting(t *testing.T) {
	opb := NewCongressVoting("dummy contract", 1, 100, common.Amount(1000000), "dummy account")
	op := Operation{
		H: Header{Type: TypeCongressVoting},
		B: opb,
	}
	hashed := op.MakeHashString()

	expected := "EtVW5hG3p4YsSzL3mgwejHvtskzYuxW8dNaM6UEm42DX"
	require.Equal(t, expected, hashed)

	err := op.IsWellFormed(common.NewTestConfig())
	require.NoError(t, err)

}

func TestOperationBodyCongressVotingResult(t *testing.T) {
	opb := NewCongressVotingResult(
		string(common.MakeHash([]byte("dummydummy"))),
		[]string{"http://www.boscoin.io/1", "http://www.boscoin.io/2"},
		string(common.MakeHash([]byte("dummydummy"))),
		[]string{"http://www.boscoin.io/3", "http://www.boscoin.io/4"},
		string(common.MakeHash([]byte("dummydummy"))),
		[]string{"http://www.boscoin.io/3", "http://www.boscoin.io/4"},
		9, 2, 3, 4,
		"dummy voting hash-0",
	)
	op := Operation{
		H: Header{Type: TypeCongressVotingResult},
		B: opb,
	}
	hashed := op.MakeHashString()

	expected := "8t6e3gZ9BUFXGUbf8qEwQfJ3Hn7yWDAatkELaiaxbcCg"
	require.Equal(t, expected, hashed)

	err := op.IsWellFormed(common.NewTestConfig())
	require.NoError(t, err)
}

func TestOperationBodyCongressVotingResultInvalidMembership(t *testing.T) {
	{ // missing Hash
		opb := NewCongressVotingResult(
			string(common.MakeHash([]byte("dummydummy"))),
			[]string{"http://www.boscoin.io/1", "http://www.boscoin.io/2"},
			string(common.MakeHash([]byte("dummydummy"))),
			[]string{"http://www.boscoin.io/3", "http://www.boscoin.io/4"},
			"",
			[]string{"http://www.boscoin.io/5", "http://www.boscoin.io/6"},
			9, 2, 3, 4,
			"dummy voting hash",
		)
		op := Operation{
			H: Header{Type: TypeCongressVotingResult},
			B: opb,
		}

		err := op.IsWellFormed(common.NewTestConfig())
		require.Error(t, err, errors.InvalidOperation)
	}

	{ // bad urls
		opb := NewCongressVotingResult(
			string(common.MakeHash([]byte("dummydummy"))),
			[]string{"http://www.boscoin.io/1", "http://www.boscoin.io/2"},
			string(common.MakeHash([]byte("dummydummy"))),
			[]string{"http://www.boscoin.io/3", "http://www.boscoin.io/4"},
			string(common.MakeHash([]byte("dummydummy"))),
			[]string{"3", "4"},
			9, 2, 3, 4,
			"dummy voting hash",
		)
		op := Operation{
			H: Header{Type: TypeCongressVotingResult},
			B: opb,
		}

		err := op.IsWellFormed(common.NewTestConfig())
		require.Error(t, err, errors.InvalidOperation)
	}

	{ // valid operation
		opb := NewCongressVotingResult(
			string(common.MakeHash([]byte("dummydummy"))),
			[]string{"http://www.boscoin.io/1", "http://www.boscoin.io/2"},
			string(common.MakeHash([]byte("dummydummy"))),
			[]string{"http://www.boscoin.io/3", "http://www.boscoin.io/4"},
			string(common.MakeHash([]byte("dummydummy"))),
			[]string{"http://www.boscoin.io/3", "http://www.boscoin.io/4"},
			9, 2, 3, 4,
			"dummy voting hash-0",
		)
		op := Operation{
			H: Header{Type: TypeCongressVotingResult},
			B: opb,
		}

		err := op.IsWellFormed(common.NewTestConfig())
		require.NoError(t, err)
	}
}
