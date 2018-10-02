package sync

import (
	"context"
	"time"

	"boscoin.io/sebak/lib/block"
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
	localNode         *node.LocalNode

	fetcher   Fetcher
	validator Validator

	workPool   *Pool
	stop       chan chan struct{}
	ctx        context.Context
	cancelFunc context.CancelFunc

	updateHighestBlock chan uint64

	logger log15.Logger
}

type SyncerOption func(s *Syncer)

func NewSyncer(st *storage.LevelDBBackend,
	nw network.Network,
	cm network.ConnectionManager,
	networkID []byte,
	localNode *node.LocalNode,
	opts ...SyncerOption) *Syncer {
	ctx, cancelFunc := context.WithCancel(context.Background())

	s := &Syncer{
		storage:           st,
		network:           nw,
		connectionManager: cm,
		networkID:         networkID,
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

	validator := NewBlockValidator(nw, st, networkID, func(v *BlockValidator) {
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

func (s *Syncer) SetSyncTargetBlock(ctx context.Context, height uint64) error {

	select {
	case s.updateHighestBlock <- height:
	case <-ctx.Done():
		return ctx.Err()
	}

	return nil
}

func (s *Syncer) loop() {
	checkc := s.afterFunc(s.checkInterval)

	height := s.lastestBlockHeight()
	syncProgress := &SyncProgress{
		StartingBlock: height,
		CurrentBlock:  height,
		HighestBlock:  height,
	}

	s.logger.Info("starting block to sync", "height", height)

	for {
		select {
		case <-checkc:
			syncProgress.HighestBlock++ // TODO(anarcher): Until work together consensus
			s.sync(syncProgress)
			checkc = s.afterFunc(s.checkInterval)
		case height := <-s.updateHighestBlock:
			if height >= syncProgress.CurrentBlock {
				syncProgress.HighestBlock = height
				s.sync(syncProgress)
			}
		case c := <-s.stop:
			close(c)
			return
		}
	}
}

func (s *Syncer) sync(p *SyncProgress) {
	var (
		currentHeight      uint64
		lastestBlockHeight = s.lastestBlockHeight()
		log                = func() {
			s.logger.Info("sync progress",
				"start", p.StartingBlock, "cur", p.CurrentBlock, "high", p.HighestBlock)
		}
	)

	if lastestBlockHeight > p.CurrentBlock {
		p.CurrentBlock = lastestBlockHeight
	}

	if p.CurrentBlock >= p.HighestBlock {
		log()
		return
	}

	for height := p.CurrentBlock + 1; height <= p.HighestBlock; height++ {
		// TryAdd for unblocking when the pool is full. Just keep syncprogress for next sync
		if s.work(height) == false {
			break
		}
		currentHeight = height
	}
	p.StartingBlock = p.CurrentBlock
	p.CurrentBlock = currentHeight
	log()
}

func (s *Syncer) work(height uint64) bool {
	ctx := s.ctx
	work := func() {
		lastHeight := s.lastestBlockHeight()
		if lastHeight > 0 && height <= lastHeight {
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
	return s.workPool.TryAdd(ctx, work)
}

func (s *Syncer) lastestBlockHeight() uint64 {
	blk, err := block.GetLatestBlock(s.storage)
	if err != nil {
		s.logger.Error("block.GetLatestBlock", "err", err)
		return 0
	}
	return blk.Height
}
