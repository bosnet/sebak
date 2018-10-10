package sync

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/network"
	"boscoin.io/sebak/lib/storage"
	"github.com/stretchr/testify/assert"
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
		var (
			ctx    = context.Background()
			syncer = tctx.syncer
			infoc  = tctx.syncInfoC
		)

		{
			bk := block.TestMakeNewBlock([]string{})
			bk.Height = uint64(1)
			bk.Save(tctx.st)
		}

		var (
			height    uint64   = 10
			cnt       uint64   = 2
			nodeAddrs []string = []string{"a", "b"}
		)

		syncer.validator = &mockValidator{
			validateFunc: func(ctx context.Context, si *SyncInfo) error {
				assert.Equal(t, len(si.NodeAddrs), 2)
				infoc <- si
				atomic.AddUint64(&cnt, 1)
				if atomic.LoadUint64(&cnt) > 10 {
					close(infoc)
				}
				return nil
			},
		}

		go func() {
			syncer.Start()
		}()

		syncer.SetSyncTargetBlock(ctx, height, nodeAddrs)

		var heights []uint64
		for si := range infoc {
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
			si.BlockHeight = bk.Height
			si.Block = &bk
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
