package sync

import (
	"time"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/network"
	"boscoin.io/sebak/lib/node"
	"boscoin.io/sebak/lib/storage"
	"github.com/inconshreveable/log15"
)

const (
	SyncPoolSize             = 300
	FetchTimeout             = 1 * time.Minute
	RetryInterval            = 10 * time.Second
	CheckBlockHeightInterval = 30 * time.Second
)

type Config struct {
	storage           *storage.LevelDBBackend
	network           network.Network
	connectionManager network.ConnectionManager
	networkID         []byte
	localNode         *node.LocalNode
	logger            log15.Logger
	commonCfg         common.Config

	SyncPoolSize             int
	FetchTimeout             time.Duration
	RetryInterval            time.Duration
	CheckBlockHeightInterval time.Duration
	Logger                   log15.Logger
}

func NewConfig(networkID []byte,
	localNode *node.LocalNode,
	st *storage.LevelDBBackend,
	nt network.Network,
	cm network.ConnectionManager,
	cfg common.Config) *Config {
	c := &Config{
		storage:           st,
		network:           nt,
		connectionManager: cm,
		logger:            log.New(log15.Ctx{"node": localNode.Alias()}),
		networkID:         networkID,
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
	s := NewSyncer(c.storage, c.network, c.connectionManager, c.networkID, c.commonCfg, func(s *Syncer) {
		s.poolSize = c.SyncPoolSize
		s.fetchTimeout = c.FetchTimeout
		s.retryInterval = c.RetryInterval
		s.checkInterval = c.CheckBlockHeightInterval
		s.logger = c.logger
	})

	c.LoggingConfig()
	return s
}

func (c *Config) LoggingConfig() {
	c.logger.Info("syncer config",
		"poolSize", c.SyncPoolSize,
		"fetchTimeout", c.FetchTimeout,
		"retryInterval", c.RetryInterval,
		"checkInterval", c.CheckBlockHeightInterval,
	)

}
