package sync

import (
	"context"
	"time"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/network"
	"boscoin.io/sebak/lib/storage"
	"github.com/inconshreveable/log15"
)

type Syncer struct {
	poolSize      int
	fetchTimeout  time.Duration
	retryInterval time.Duration
	checkInterval time.Duration

	afterFunc AfterFunc

	storage           *storage.LevelDBBackend
	network           network.Network
	connectionManager network.ConnectionManager
	networkID         []byte
	commonCfg         common.Config

	fetcher   Fetcher
	validator Validator

	workPool   *Pool
	stop       chan chan struct{}
	ctx        context.Context
	cancelFunc context.CancelFunc

	logger log15.Logger
}

type SyncerOption func(s *Syncer)

func NewSyncer(st *storage.LevelDBBackend,
	nw network.Network,
	cm network.ConnectionManager,
	networkID []byte,
	cfg common.Config, opts ...SyncerOption) *Syncer {
	ctx, cancelFunc := context.WithCancel(context.Background())

	s := &Syncer{
		storage:           st,
		network:           nw,
		connectionManager: cm,
		networkID:         networkID,
		commonCfg:         cfg,

		poolSize:      SyncPoolSize,
		fetchTimeout:  FetchTimeout,
		retryInterval: RetryInterval,
		checkInterval: CheckBlockHeightInterval,

		afterFunc: time.After,

		stop:       make(chan chan struct{}),
		ctx:        ctx,
		cancelFunc: cancelFunc,

		logger: NopLogger(),
	}

	for _, opt := range opts {
		opt(s)
	}

	fetcher := NewBlockFetcher(nw, cm, st, func(f *BlockFetcher) {
		f.fetchTimeout = s.fetchTimeout
		f.logger = s.logger
	})
	s.fetcher = fetcher

	validator := NewBlockValidator(nw, st, networkID, cfg, func(v *BlockValidator) {
		v.logger = s.logger
	})
	s.validator = validator

	return s
}

func (s *Syncer) Stop() error {
	s.cancelFunc()
	c := make(chan struct{})
	s.stop <- c
	<-c
	s.workPool.Finish()
	s.logger.Info("Stopped syncer")
	return nil
}

func (s *Syncer) Start() error {
	s.logger.Info("Starting syncer")
	s.workPool = NewPool(s.poolSize)
	s.loop()
	return nil
}

func (s *Syncer) loop() {
	checkc := s.afterFunc(s.checkInterval)
	blockHeight := uint64(1)
	for {
		select {
		case <-checkc:
			blockHeight = s.syncBlockHeight(blockHeight)
			checkc = s.afterFunc(s.checkInterval)
		case c := <-s.stop:
			close(c)
			return
		}
	}
}

func (s *Syncer) syncBlockHeight(height uint64) uint64 {
	blk, err := block.GetLatestBlock(s.storage)
	if err != nil {
		s.logger.Error("checkBlockHeight", "err", err)
	}
	newHeight := blk.Height + 1
	if newHeight > height {
		s.work(newHeight)
		s.logger.Info("Starting sync", "height", newHeight)
		return newHeight
	}
	return height
}

func (s *Syncer) work(height uint64) {
	ctx := s.ctx
	work := func() {
		currHeight := s.currBlockHeight()
		if currHeight > 0 && height <= currHeight {
			s.logger.Info("This height has already synced", "height", height)
			return
		}

		var (
			syncInfo = &SyncInfo{BlockHeight: height}
			err      error
		)

	L:
		for {
			select {
			case <-ctx.Done():
				break L
			default:
				syncInfo, _ = s.fetcher.Fetch(ctx, syncInfo)
				err = s.validator.Validate(ctx, syncInfo)
				if err != nil {
					s.logger.Error("validate failure", "err", err)
					continue
				}
				break L
			}
		}
		if err != nil {
			s.logger.Info("Stop sync work", "height", height, "err", err)
		} else {
			s.logger.Info("Done sync work", "height", height)
		}
	}
	s.workPool.Add(ctx, work)
}

func (s *Syncer) currBlockHeight() uint64 {
	blk, err := block.GetLatestBlock(s.storage)
	if err != nil {
		s.logger.Error("block.GetLatestBlock", "err", err)
		return 0
	}
	return blk.Height
}
