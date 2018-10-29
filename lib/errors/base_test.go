package errors

import (
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/rlp"
	"github.com/stretchr/testify/require"
)

func TestErrorsClone(t *testing.T) {
	require.Equal(t, BlockAlreadyExists, BlockAlreadyExists)

	e := BlockAlreadyExists
	e0 := BlockAlreadyExists.Clone()
	require.NotEqual(t, fmt.Sprintf("%p", e), fmt.Sprintf("%p", e0))

	{
		e.Code = 200
		require.NotEqual(t, e.Code, e0.Code)
	}

	{
		e0.SetData("showme", "killme")
		require.NotEqual(t, e.Data, e0.Data)
	}
}

func TestErrorsRLP(t *testing.T) {
	{
		_, err := rlp.EncodeToBytes(BlockAlreadyExists)
		require.NoError(t, err)
	}

	{ // with `SetData()`, the rlp encoded value must be different
		encoded, err := rlp.EncodeToBytes(BlockAlreadyExists)
		require.NoError(t, err)

		e := BlockAlreadyExists.Clone()
		e.SetData("findme", "killme")
		encoded0, err := rlp.EncodeToBytes(e)
		require.NoError(t, err)
		require.NotEqual(t, encoded, encoded0)
	}
}
