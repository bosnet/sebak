package sync

import (
	"context"
	"testing"
	"time"

	"boscoin.io/sebak/lib/block"
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

		var (
			height    uint64   = 10
			nodeAddrs []string = []string{"a", "b"}
		)

		syncer.validator = &mockValidator{
			validateFunc: func(ctx context.Context, si *SyncInfo) error {
				assert.Equal(t, len(si.NodeAddrs), 2)
				infoc <- si
				return nil
			},
		}

		go func() {
			syncer.Start()
		}()

		syncer.SetSyncTargetBlock(ctx, height, nodeAddrs)

		var heights []uint64
		for si := range infoc {
			heights = append(heights, si.Height)
			if len(heights) >= 9 {
				close(infoc)
			}
		}
		require.Equal(t, len(heights), 9)

		progress, err := syncer.SyncProgress(ctx)
		require.NoError(t, err)
		require.Equal(t, progress.StartingBlock, uint64(2))
		require.Equal(t, progress.CurrentBlock, height)
		require.Equal(t, progress.HighestBlock, height)

	}
	SyncerTest(t, fn)
}

func SyncerTest(t *testing.T, fn func(*SyncerTestContext)) {
	st := block.InitTestBlockchain()
	defer st.Close()

	tickc := make(chan time.Time)
	infoc := make(chan *SyncInfo)

	fetcher := &mockFetcher{
		fetchFunc: func(ctx context.Context, si *SyncInfo) (*SyncInfo, error) {
			bk := block.TestMakeNewBlock([]string{})
			bk.Height = si.Height
			si.Height = bk.Height
			si.Block = &bk
			return si, nil
		},
	}
	validator := &mockValidator{
		validateFunc: func(ctx context.Context, si *SyncInfo) error {
			infoc <- si
			return nil
		},
	}

	syncer := NewSyncer(fetcher, validator, st, func(s *Syncer) {
		s.afterFunc = func(d time.Duration) <-chan time.Time {
			return tickc
		}
	})
	defer syncer.Stop()

	ctx := &SyncerTestContext{
		t:         t,
		st:        st,
		syncer:    syncer,
		tickC:     tickc,
		syncInfoC: infoc,
	}

	fn(ctx)
}
