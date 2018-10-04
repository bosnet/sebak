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

type SyncerTestContext struct {
	t         *testing.T
	st        *storage.LevelDBBackend
	syncer    *Syncer
	tickC     chan time.Time
	syncInfoC chan *SyncInfo
}

func TestSyncerSetSyncTarget(t *testing.T) {
	fn := func(tctx *SyncerTestContext) {
		ctx := context.Background()
		syncer := tctx.syncer

		{
			bk := block.TestMakeNewBlock([]string{})
			bk.Height = uint64(1)
			bk.Save(tctx.st)
		}

		go func() {
			syncer.Start()
		}()

		height := uint64(10)
		syncer.SetSyncTargetBlock(ctx, height)
		sp, err := syncer.SyncProgress(ctx)
		require.Nil(t, err)

		var heights []uint64
		for i := sp.StartingBlock; i <= sp.CurrentBlock; i++ {
			si := <-tctx.syncInfoC
			heights = append(heights, si.BlockHeight)
		}
		require.Equal(t, len(heights), 9)
	}
	SyncerTest(t, fn)
}

func SyncerTest(t *testing.T, fn func(*SyncerTestContext)) {
	st := storage.NewTestStorage()
	defer st.Close()
	_, nw, localNode := network.CreateMemoryNetwork(nil)
	cm := &mockConnectionManager{}
	networkID := []byte("test-network")

	tickc := make(chan time.Time)
	infoc := make(chan *SyncInfo)

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
			infoc <- si
			return nil
		},
	}
	syncer.afterFunc = func(d time.Duration) <-chan time.Time {
		return tickc
	}

	ctx := &SyncerTestContext{
		t:         t,
		st:        st,
		syncer:    syncer,
		tickC:     tickc,
		syncInfoC: infoc,
	}

	fn(ctx)
}
