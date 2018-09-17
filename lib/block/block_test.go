package block

import (
	"math/rand"
	"testing"
	"time"

	"boscoin.io/sebak/lib/storage"
	"github.com/stretchr/testify/require"
)

func TestBlockConfirmedOrdering(t *testing.T) {
	st, _ := storage.NewTestMemoryLevelDBBackend()

	var inserted []Block
	for i := 0; i < 10; i++ {
		bk := TestMakeNewBlock([]string{})
		bk.Height = uint64(i)
		require.Nil(t, bk.Save(st))
		inserted = append(inserted, bk)
	}

	{ // reverse = false
		var fetched []Block
		iterFunc, closeFunc := GetBlocksByConfirmed(st, storage.NewDefaultListOptions(false, nil, 10))
		for {
			t, hasNext, _ := iterFunc()
			if !hasNext {
				break
			}
			fetched = append(fetched, t)
		}
		closeFunc()

		require.Equal(t, len(inserted), len(fetched))
		for i, b := range inserted {
			require.Equal(t, b.Hash, fetched[i].Hash)

			s, _ := b.Serialize()
			rs, _ := fetched[i].Serialize()
			require.Equal(t, s, rs)
		}
	}

	{ // reverse = true
		var fetched []Block
		iterFunc, closeFunc := GetBlocksByConfirmed(st, storage.NewDefaultListOptions(true, nil, 10))
		for {
			t, hasNext, _ := iterFunc()
			if !hasNext {
				break
			}
			fetched = append(fetched, t)
		}
		closeFunc()

		require.Equal(t, len(inserted), len(fetched))
		for i, b := range inserted {
			require.Equal(t, b.Hash, fetched[len(fetched)-1-i].Hash)

			s, _ := b.Serialize()
			rs, _ := fetched[len(fetched)-1-i].Serialize()
			require.Equal(t, s, rs)
		}
	}
}

func TestBlockHeightOrdering(t *testing.T) {
	st, _ := storage.NewTestMemoryLevelDBBackend()

	// save Block, but Height will be shuffled
	numberOfBlocks := 10
	inserted := make([]Block, numberOfBlocks)

	r := rand.New(rand.NewSource(time.Now().Unix()))
	for _, i := range r.Perm(numberOfBlocks) {
		bk := TestMakeNewBlock([]string{})
		bk.Height = uint64(i)
		require.Nil(t, bk.Save(st))
		inserted[i] = bk
	}

	{
		var fetched []Block
		for i := 0; i < numberOfBlocks; i++ {
			b, err := GetBlockByHeight(st, uint64(i))
			require.Nil(t, err)
			fetched = append(fetched, b)
		}

		require.Equal(t, len(inserted), len(fetched))
		for i, b := range inserted {
			require.Equal(t, b.Hash, fetched[i].Hash)

			s, _ := b.Serialize()
			rs, _ := fetched[i].Serialize()
			require.Equal(t, s, rs)
		}
	}
}
