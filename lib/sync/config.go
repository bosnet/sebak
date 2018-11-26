package sync

import (
	"time"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/network"
	"boscoin.io/sebak/lib/node"
	"boscoin.io/sebak/lib/storage"
	"boscoin.io/sebak/lib/transaction"
	"github.com/inconshreveable/log15"
)

const (
	SyncPoolSize             uint64 = 300
	FetchTimeout                    = 1 * time.Minute
	RetryInterval                   = 10 * time.Second
	CheckBlockHeightInterval        = 30 * time.Second
	CheckPrevBlockInterval          = 30 * time.Second
)

type Config struct {
	storage           *storage.LevelDBBackend
	network           network.Network
	connectionManager network.ConnectionManager
	tp                *transaction.Pool
	localNode         *node.LocalNode
	logger            log15.Logger
	commonCfg         common.Config

	SyncPoolSize             uint64
	FetchTimeout             time.Duration
	RetryInterval            time.Duration
	CheckBlockHeightInterval time.Duration
	CheckPrevBlockInterval   time.Duration
}

func NewConfig(localNode *node.LocalNode,
	st *storage.LevelDBBackend,
	nt network.Network,
	cm network.ConnectionManager,
	tp *transaction.Pool,
	cfg common.Config) *Config {
	c := &Config{
		storage:           st,
		network:           nt,
		connectionManager: cm,
		tp:                tp,
		logger:            log.New(log15.Ctx{"node": localNode.Alias()}),
		commonCfg:         cfg,
		localNode:         localNode,

		SyncPoolSize:             SyncPoolSize,
		FetchTimeout:             FetchTimeout,
		RetryInterval:            RetryInterval,
		CheckBlockHeightInterval: CheckBlockHeightInterval,
	}
	return c
}

func (c *Config) NewSyncer() *Syncer {
	f := c.NewFetcher()
	v := c.NewValidator()
	s := NewSyncer(f, v, c.storage, func(s *Syncer) {
		s.poolSize = c.SyncPoolSize
		s.checkInterval = c.CheckBlockHeightInterval
		s.logger = c.logger.New("submodule", "syncer")
	})

	c.LoggingConfig()
	return s
}

func (c *Config) NewFetcher() Fetcher {
	f := NewBlockFetcher(
		c.network,
		c.connectionManager,
		c.storage,
		c.localNode,
		func(f *BlockFetcher) {
			f.fetchTimeout = c.FetchTimeout
			f.logger = c.logger.New("submodule", "fetcher")
		},
	)
	return f
}

func (c *Config) NewValidator() Validator {
	v := NewBlockValidator(
		c.network,
		c.storage,
		c.tp,
		c.commonCfg,
		func(v *BlockValidator) {
			v.prevBlockWaitTimeout = c.CheckPrevBlockInterval
			v.logger = c.logger.New("submodule", "validator")
		})
	return v
}

func (c *Config) LoggingConfig() {
	c.logger.Info("syncer config",
		"poolSize", c.SyncPoolSize,
		"fetchTimeout", c.FetchTimeout,
		"retryInterval", c.RetryInterval,
		"checkInterval", c.CheckBlockHeightInterval,
		"checkPrevBlockInterval", c.CheckPrevBlockInterval,
	)
}
