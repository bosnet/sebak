package sync

import (
	"context"
	"testing"
	"time"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/network"
	"boscoin.io/sebak/lib/storage"
	"github.com/stretchr/testify/require"
)

func TestSyncer(t *testing.T) {
	st := storage.NewTestStorage()
	defer st.Close()
	_, nw, _ := network.CreateMemoryNetwork(nil)
	cm := &mockConnectionManager{}

	tickc := make(chan time.Time)
	blockInfoC := make(chan *BlockInfo)

	syncer := NewSyncer(st, nw, cm)
	defer syncer.Stop()

	syncer.fetcher = &mockFetcher{
		fetchFunc: func(ctx context.Context, height uint64) (*BlockInfo, error) {
			bk := block.TestMakeNewBlock([]string{})
			bk.Height = height
			bi := &BlockInfo{
				BlockHeight: height,
				Block:       &bk,
			}
			return bi, nil
		},
	}
	syncer.validator = &mockValidator{
		validateFunc: func(ctx context.Context, bi *BlockInfo) error {
			blockInfoC <- bi
			return nil
		},
	}
	syncer.afterFunc = func(d time.Duration) <-chan time.Time {
		return tickc
	}

	go func() {
		syncer.Start()
	}()

	{
		bk := block.TestMakeNewBlock([]string{})
		bk.Height = uint64(1)
		require.Nil(t, bk.Save(st))

		tickc <- time.Time{}
		bi := <-blockInfoC
		require.NotNil(t, bi.Block)
		require.Equal(t, bi.BlockHeight, uint64(2))
	}
}
