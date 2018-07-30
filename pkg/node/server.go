package node

import (
	"boscoin.io/sebak/pkg/catchup"
	"boscoin.io/sebak/pkg/consensus"
	"boscoin.io/sebak/pkg/core"
	"boscoin.io/sebak/pkg/network"
	"boscoin.io/sebak/pkg/network/tcp"
	"boscoin.io/sebak/pkg/rawdb"
	"boscoin.io/sebak/pkg/support/logger"
	"boscoin.io/sebak/pkg/wire"
	"boscoin.io/sebak/pkg/wire/message"
	"fmt"
	"net/url"
	"sync"
	"time"
)

type server struct {
	lock   sync.Mutex
	logger *logger.Logger

	running bool
	chain   *core.BlockChain

	config         *Config
	networkManager network.Manager
}

func newServer(config *Config) *server {
	log := logger.NewLogger("server")
	log.Info("msg", "open database", "dir", config.ChainDbDir)

	blockDb, err := rawdb.NewLevelDb(config.ChainDbDir)
	if err != nil {
		panic(err)
	}

	chain := core.NewBlockChain(blockDb)
	protocol := message.RegisterAllMessages(wire.NewMsgpackProtocol())

	networkManager := tcp.NewTcpNetwork(&tcp.Params{
		ListenAddresses: config.ListenAddresses,
		PubKey:          message.PeerId(config.KeyPair.Address()),
		Protocol:        protocol,
	})

	networkManager.AddReceiver(catchup.NewBlockRequestPool(chain, networkManager))
	networkManager.AddReceiver(consensus.NewIsaacReceiver())

	return &server{
		logger:         log,
		chain:          chain,
		config:         config,
		networkManager: networkManager,
	}
}

func (o *server) Start() error {
	o.lock.Lock()
	defer o.lock.Unlock()

	o.running = true

	o.networkManager.Start()

	for _, peerAddress := range o.config.Validators {
		go o.connectTo(peerAddress)
	}

	return nil
}

func (o *server) Stop() {
	o.networkManager.Stop()
}

func (o *server) connectTo(peerAddress *url.URL) {
	backOff := time.Millisecond * 1000
	timer := time.NewTimer(backOff)

	for {
		select {
		case <-timer.C:
			if err := o.networkManager.Connect(peerAddress); err != nil {
				o.logger.Warn("msg", fmt.Sprintf("Failed to connect to peer. retry after %s", backOff), "address", peerAddress)
				backOff = backOff * 5 / 4
				if backOff > time.Second*10 {
					backOff = time.Second * 10
				}
				timer.Reset(backOff)
			}
		}
	}
}
