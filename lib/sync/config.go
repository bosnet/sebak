package sync

import (
	"time"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/network"
	"boscoin.io/sebak/lib/node"
	"boscoin.io/sebak/lib/node/runner"
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
	WatchInterval                   = 5 * time.Second
)

type Config struct {
	storage           *storage.LevelDBBackend
	connectionManager network.ConnectionManager
	tp                *transaction.Pool
	localNode         *node.LocalNode
	nodelist          *NodeList
	logger            log15.Logger
	commonCfg         common.Config

	SyncPoolSize             uint64
	FetchTimeout             time.Duration
	RetryInterval            time.Duration
	CheckBlockHeightInterval time.Duration
	CheckPrevBlockInterval   time.Duration
	WatchInterval            time.Duration
}

func NewConfig(localNode *node.LocalNode,
	st *storage.LevelDBBackend,
	cm network.ConnectionManager,
	tp *transaction.Pool,
	cfg common.Config) (*Config, error) {
	c := &Config{
		storage:           st,
		connectionManager: cm,
		tp:                tp,
		logger:            log.New(log15.Ctx{"node": localNode.Alias()}),
		commonCfg:         cfg,
		localNode:         localNode,
		nodelist:          &NodeList{},

		SyncPoolSize:             SyncPoolSize,
		FetchTimeout:             FetchTimeout,
		RetryInterval:            RetryInterval,
		CheckBlockHeightInterval: CheckBlockHeightInterval,
	}
	commonAccountAddress, err := c.commonAccountAddress()
	if err != nil {
		return nil, err
	}
	c.commonCfg.CommonAccountAddress = commonAccountAddress
	return c, nil
}

func (c *Config) NewSyncer() *Syncer {
	f := c.NewFetcher()
	v := c.NewValidator()
	s := NewSyncer(f, v, c.storage, func(s *Syncer) {
		s.nodelist = c.nodelist
		s.poolSize = c.SyncPoolSize
		s.checkInterval = c.CheckBlockHeightInterval
		s.logger = c.logger.New("submodule", "syncer")
	})

	c.LoggingConfig()
	return s
}

func (c *Config) NewFetcher() Fetcher {
	client := c.NewHTTP2Client()

	f := NewBlockFetcher(
		c.connectionManager,
		client,
		c.storage,
		c.localNode,
		func(f *BlockFetcher) {
			f.fetchTimeout = c.FetchTimeout
			f.retryInterval = c.RetryInterval
			f.logger = c.logger.New("submodule", "fetcher")
		},
	)
	return f
}

func (c *Config) NewValidator() Validator {
	v := NewBlockValidator(
		c.storage,
		c.tp,
		c.commonCfg,
		func(v *BlockValidator) {
			v.prevBlockWaitTimeout = c.CheckPrevBlockInterval
			v.logger = c.logger.New("submodule", "validator")
		})
	return v
}

func (c *Config) NewWatcher(s SyncController) *Watcher {
	c.logger.Info("watcher config", "watchInterval", c.WatchInterval)

	client := c.NewHTTP2Client()
	w := NewWatcher(
		s, client,
		c.connectionManager,
		c.storage,
		c.localNode,
		func(w *Watcher) {
			w.interval = c.WatchInterval
		},
	)
	w.SetLogger(c.logger.New("submodule", "watcher"))
	return w
}

func (c *Config) NewHTTP2Client() *common.HTTP2Client {
	client, err := common.NewHTTP2Client(c.FetchTimeout, 0, true)
	if err != nil {
		c.logger.Error("make http2 client error!", "err", err)
		panic(err) // It's an unrecoverable error not to make client when starting syncer / node
	}
	return client
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

func (c *Config) commonAccountAddress() (string, error) {
	commonAccount, err := runner.GetCommonAccount(c.storage)
	if err != nil {
		return "", err
	}

	return commonAccount.Address, nil
}
