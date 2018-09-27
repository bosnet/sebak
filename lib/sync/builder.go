package sync

import (
	"time"

	"boscoin.io/sebak/lib/network"
	"boscoin.io/sebak/lib/node"
	"boscoin.io/sebak/lib/storage"

	"github.com/inconshreveable/log15"
)

const (
	MaxFetcher               = 10
	MaxValidator             = 20
	FetchTimeout             = 1 * time.Minute
	ValidationTimeout        = 2 * time.Minute
	RetryInterval            = 10 * time.Second
	CheckBlockHeightInterval = 1 * time.Second
)

type Builder struct {
	MaxFetcher               int
	MaxValidator             int
	FetchTimeout             time.Duration
	ValidationTimeout        time.Duration
	RetryInterval            time.Duration
	CheckBlockHeightInterval time.Duration

	localNode         *node.LocalNode
	network           network.Network
	connectionManager network.ConnectionManager
	storage           *storage.LevelDBBackend

	logger log15.Logger
}

func NewBuilder(localNode *node.LocalNode, ldb *storage.LevelDBBackend, network network.Network, connectionManager network.ConnectionManager) Builder {
	return Builder{
		MaxFetcher:               MaxFetcher,
		MaxValidator:             MaxValidator,
		FetchTimeout:             FetchTimeout,
		ValidationTimeout:        ValidationTimeout,
		RetryInterval:            RetryInterval,
		CheckBlockHeightInterval: CheckBlockHeightInterval,

		network:           network,
		connectionManager: connectionManager,
		storage:           ldb,

		logger: log.New(log15.Ctx{"node": localNode.Alias()}),
	}
}

func (b Builder) BlockFullFetchers() []Fetcher {
	var fs []Fetcher

	for i := 0; i < b.MaxFetcher; i++ {
		fs = append(fs, NewBlockFullFetcher(b.network, b.connectionManager, func(f *BlockFullFetcher) {
			f.fetchTimeout = b.FetchTimeout
			f.logger = b.logger.New(log15.Ctx{"component": "fetcher"})
		}))
	}

	return fs
}

func (b Builder) BlockValidators() []Validator {
	var vs []Validator

	for i := 0; i < b.MaxValidator; i++ {
		vs = append(vs, NewBlockValidator(b.network, b.storage, func(v *BlockValidator) {
			v.validationTimeout = b.ValidationTimeout
			v.logger = b.logger.New(log15.Ctx{"component": "validator"})
		}))
	}

	return vs
}

func (b Builder) Manager() *Manager {
	var (
		ps []Processor
		cs []Consumer
	)
	for _, p := range b.BlockFullFetchers() {
		ps = append(ps, p)
	}
	for _, c := range b.BlockValidators() {
		cs = append(cs, c)
	}

	fetcher := NewProcessors(ps...)
	validator := NewConsumers(cs...)

	m := &Manager{
		fetcherLayer:    fetcher,
		validationLayer: validator,

		retryInterval: b.RetryInterval,
		checkInterval: b.CheckBlockHeightInterval,

		afterFunc: time.After,

		storage:  b.storage,
		stopLoop: make(chan chan struct{}),
		stopResp: make(chan chan struct{}),
		cancel:   make(chan chan struct{}),

		logger: b.logger.New(log15.Ctx{"component": "manager"}),
	}

	Pipeline(m, fetcher)
	Pipeline(fetcher, validator)

	return m
}
