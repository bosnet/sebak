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
	_, nw, localNode := network.CreateMemoryNetwork(nil)
	cm := &mockConnectionManager{}
	networkID := []byte("test-network")

	tickc := make(chan time.Time)
	infoC := make(chan *SyncInfo)

	syncer := NewSyncer(st, nw, cm, networkID, localNode)
	defer syncer.Stop()

	syncer.fetcher = &mockFetcher{
		fetchFunc: func(ctx context.Context, si *SyncInfo) (*SyncInfo, error) {
			bk := block.TestMakeNewBlock([]string{})
			bk.Height = si.BlockHeight
			si = &SyncInfo{
				BlockHeight: bk.Height,
				Block:       &bk,
			}
			return si, nil
		},
	}
	syncer.validator = &mockValidator{
		validateFunc: func(ctx context.Context, si *SyncInfo) error {
			infoC <- si
			return nil
		},
	}
	syncer.afterFunc = func(d time.Duration) <-chan time.Time {
		return tickc
	}

	{
		bk := block.TestMakeNewBlock([]string{})
		bk.Height = uint64(1)
		require.Nil(t, bk.Save(st))
	}

	go func() {
		syncer.Start()
	}()

	{
		tickc <- time.Time{}
		si := <-infoC
		require.NotNil(t, si.Block)
		require.Equal(t, si.BlockHeight, uint64(2))
	}
}
