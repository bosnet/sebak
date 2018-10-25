package storage

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	"boscoin.io/sebak/lib/errors"
)

func TestBatchBackendNew(t *testing.T) {
	st := NewTestStorage()
	defer st.Close()

	fetched := map[int]string{}
	key := "showme"
	input := map[int]string{
		90: "99",
		91: "91",
		92: "92",
	}

	bt, err := st.OpenBatch()
	require.NoError(t, err)

	{ // `Get` failed in both
		{ // in normal LeveldbBatch
			err := st.Get(key, &fetched)
			require.Equal(t, errors.ErrorStorageRecordDoesNotExist, err)
		}

		{ // in BatchBackend
			err := bt.Get(key, &fetched)
			require.Equal(t, errors.ErrorStorageRecordDoesNotExist, err)
		}
	}

	{ // `New` in BatchBackend, but it does not stored in LeveldbBatch
		{ // in BatchBackend
			err := bt.New(key, input)
			require.NoError(t, err)
		}

		{ // in normal LeveldbBatch
			err := st.Get(key, &fetched)
			require.Equal(t, errors.ErrorStorageRecordDoesNotExist, err)
		}
	}

	{ // `Get` must return the value of `New` in BatchBackend
		err := bt.Get(key, &fetched)
		require.NoError(t, err)

		require.True(t, reflect.DeepEqual(input, fetched))
	}

	{ // `New` must be failed because already `New`ed
		err = bt.New(key, input)
		require.Equal(t, errors.ErrorStorageRecordAlreadyExists.Code, err.(*errors.Error).Code)
	}

	{ // `Commit` batch, it must be stored in LeveldbBatch
		err := bt.Commit()
		require.NoError(t, err)

		err = st.Get(key, &fetched)
		require.NoError(t, err)
		require.True(t, reflect.DeepEqual(input, fetched))
	}
}

func TestBatchBackendDelete(t *testing.T) {
	st := NewTestStorage()
	defer st.Close()

	fetched := map[int]string{}
	key := "showme"
	input := map[int]string{
		90: "99",
		91: "91",
		92: "92",
	}

	{
		err := st.New(key, input)
		require.NoError(t, err)
	}

	bt, _ := st.OpenBatch()

	// `Delete` must be failed because already `New`ed
	bt.Remove(key)

	{ // in LeveldbBatch still have data
		err := st.Get(key, &fetched)
		require.NoError(t, err)

		require.True(t, reflect.DeepEqual(input, fetched))
	}

	err := bt.Commit()
	require.NoError(t, err)

	{ // after `Commit`, it must be removed in LeveldbBatch and BatchBackend
		err = bt.Get(key, &fetched)
		require.Equal(t, errors.ErrorStorageRecordDoesNotExist, err)

		err = st.Get(key, &fetched)
		require.Equal(t, errors.ErrorStorageRecordDoesNotExist, err)
	}
}
