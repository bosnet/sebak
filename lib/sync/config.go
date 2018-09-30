package sync

import (
	"time"

	"boscoin.io/sebak/lib/network"
	"boscoin.io/sebak/lib/node"
	"boscoin.io/sebak/lib/storage"
	"github.com/inconshreveable/log15"
)

const (
	SyncPoolSize             = 300
	FetchTimeout             = 1 * time.Minute
	RetryInterval            = 10 * time.Second
	CheckBlockHeightInterval = 500 * time.Millisecond
)

type Config struct {
	storage           *storage.LevelDBBackend
	network           network.Network
	connectionManager network.ConnectionManager
	logger            log15.Logger

	SyncPoolSize             int
	FetchTimeout             time.Duration
	RetryInterval            time.Duration
	CheckBlockHeightInterval time.Duration
	Logger                   log15.Logger
}

func NewConfig(localNode *node.LocalNode, st *storage.LevelDBBackend, nt network.Network, cm network.ConnectionManager) *Config {
	c := &Config{
		storage:           st,
		network:           nt,
		connectionManager: cm,
		logger:            log.New(log15.Ctx{"node": localNode.Alias()}),

		SyncPoolSize:             SyncPoolSize,
		FetchTimeout:             FetchTimeout,
		RetryInterval:            RetryInterval,
		CheckBlockHeightInterval: CheckBlockHeightInterval,
	}
	return c
}

func (c *Config) NewSyncer() *Syncer {
	s := NewSyncer(c.storage, c.network, c.connectionManager, func(s *Syncer) {
		s.poolSize = c.SyncPoolSize
		s.fetchTimeout = c.FetchTimeout
		s.retryInterval = c.RetryInterval
		s.checkInterval = c.CheckBlockHeightInterval
		s.logger = c.logger
	})
	return s
}
