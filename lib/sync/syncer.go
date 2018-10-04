package sync

import (
	"context"
	"fmt"
	"time"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/network"
	"boscoin.io/sebak/lib/node"
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
	localNode         *node.LocalNode

	fetcher   Fetcher
	validator Validator

	workPool   *Pool
	stop       chan chan struct{}
	ctx        context.Context
	cancelFunc context.CancelFunc

	updateHighestBlock chan uint64
	getSyncProgress    chan chan *SyncProgress

	logger log15.Logger
}

type SyncerOption func(s *Syncer)

func NewSyncer(st *storage.LevelDBBackend,
	nw network.Network,
	cm network.ConnectionManager,
	networkID []byte,
	cfg common.Config, opts ...SyncerOption) *Syncer {
	localNode *node.LocalNode,
	opts ...SyncerOption) *Syncer {
	ctx, cancelFunc := context.WithCancel(context.Background())

	s := &Syncer{
		storage:           st,
		network:           nw,
		connectionManager: cm,
		networkID:         networkID,
		commonCfg:         cfg,
		localNode:         localNode,

		poolSize:      SyncPoolSize,
		fetchTimeout:  FetchTimeout,
		retryInterval: RetryInterval,
		checkInterval: CheckBlockHeightInterval,

		afterFunc: time.After,

		stop:       make(chan chan struct{}),
		ctx:        ctx,
		cancelFunc: cancelFunc,

		updateHighestBlock: make(chan uint64),
		getSyncProgress:    make(chan chan *SyncProgress),

		logger: NopLogger(),
	}

	for _, opt := range opts {
		opt(s)
	}

	fetcher := NewBlockFetcher(nw, cm, st, localNode, func(f *BlockFetcher) {
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
	s.logger.Info("stopped syncer")
	return nil
}

func (s *Syncer) Start() error {
	s.logger.Info("starting syncer")
	s.workPool = NewPool(s.poolSize)
	s.loop()
	return nil
}

func (s *Syncer) SetSyncTargetBlock(ctx context.Context, height uint64) error {
	select {
	case s.updateHighestBlock <- height:
	case <-ctx.Done():
		return ctx.Err()
	}

	return nil
}

func (s *Syncer) SyncProgress(ctx context.Context) (*SyncProgress, error) {
	c := make(chan *SyncProgress, 1)
	select {
	case s.getSyncProgress <- c:
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	select {
	case sp := <-c:
		return sp, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (s *Syncer) loop() {
	checkc := s.afterFunc(s.checkInterval)

	height := s.latestBlockHeight()
	syncProgress := &SyncProgress{
		StartingBlock: height,
		CurrentBlock:  height,
		HighestBlock:  height,
	}

	s.logger.Info("starting block to sync", "height", height)

	for {
		select {
		case <-checkc:
			s.logger.Debug("check interval", "checkInterval", s.checkInterval)
			syncProgress.HighestBlock++ // TODO(anarcher): Until work together consensus
			s.sync(syncProgress)
			checkc = s.afterFunc(s.checkInterval)
		case height := <-s.updateHighestBlock:
			s.logger.Debug("update highest Height", "height", height)
			if height > syncProgress.CurrentBlock {
				syncProgress.HighestBlock = height
				s.sync(syncProgress)
			}
		case c := <-s.getSyncProgress:
			c <- syncProgress.Clone()
		case c := <-s.stop:
			close(c)
			return
		}
	}
}

func (s *Syncer) sync(p *SyncProgress) {
	var (
		startHeight       = p.CurrentBlock + 1
		currentHeight     = p.CurrentBlock
		highestHeight     = p.HighestBlock
		latestBlockHeight = s.latestBlockHeight()
		log               = func(msg string) {
			if msg == "" {
				msg = fmt.Sprintf("sync progress")
			}
			s.logger.Info(msg,
				"start", p.StartingBlock, "cur", p.CurrentBlock, "high", p.HighestBlock)
		}
	)

	if latestBlockHeight > currentHeight {
		startHeight = latestBlockHeight + 1
	}
	if startHeight > highestHeight {
		p.StartingBlock = latestBlockHeight + 1
		p.CurrentBlock = currentHeight
		log("sync progress skip: start height is over or equal than highest (requested) height")
		return
	}

	for height := startHeight; height <= highestHeight; height++ {
		s.logger.Info("work height", "height", height)
		// TryAdd for unblocking when the pool is full. Just keep syncprogress for next sync
		if s.work(height) == false {
			break
		}
		currentHeight = height
	}
	p.StartingBlock = startHeight
	p.CurrentBlock = currentHeight

	log("")
}

func (s *Syncer) work(height uint64) bool {
	ctx := s.ctx
	work := func() {
		latestHeight := s.latestBlockHeight()
		if latestHeight > 0 && height <= latestHeight {
			s.logger.Info("this height has already synced", "height", height)
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
				syncInfo, err = s.fetcher.Fetch(ctx, syncInfo)
				if err != nil {
					if err != context.Canceled {
						s.logger.Error("fetch failure", "err", err)
					}
				}
				err = s.validator.Validate(ctx, syncInfo)
				if err != nil {
					if err != context.Canceled {
						s.logger.Error("validate failure", "err", err)
					}
					continue
				}
				break L
			}
		}
		if err != nil {
			if err != context.Canceled {
				s.logger.Error("stop sync work", "height", height, "err", err)
			} else {
				s.logger.Debug("stop sync work", "height", height, "err", err)
			}
		} else {
			s.logger.Info("done sync work", "height", height)
		}
	}
	return s.workPool.TryAdd(ctx, work)
}

func (s *Syncer) latestBlockHeight() uint64 {
	blk, err := block.GetLatestBlock(s.storage)
	if err != nil {
		s.logger.Error("block.GetLatestBlock", "err", err)
		return 0
	}
	return blk.Height
}
