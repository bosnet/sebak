package catchup

import (
	"boscoin.io/sebak/pkg/core"
	"boscoin.io/sebak/pkg/network"
	"boscoin.io/sebak/pkg/support/logger"
	"boscoin.io/sebak/pkg/wire/message"
	"sync"
	"time"
)

const (
	blockSyncInterval              = time.Second
	blockSyncMaxConcurrentRequests = 100
)

type blockRequestState int8

type blockRequest struct {
	height    uint64
	peerId    message.PeerId
	state     blockRequestState
	startTime time.Time
}

type blockRequestPool struct {
	lock   sync.Mutex
	logger *logger.Logger

	startTime      time.Time
	chain          *core.BlockChain
	networkManager network.Manager

	peers           map[message.PeerId]*peer
	requestedBlocks map[uint64]*blockRequest
	lowestHeight    uint64
	highestHeight   uint64
	latestHeight    uint64
}

type peer struct {
	id       message.PeerId
	height   uint64
	pendings int32
	errors   int32
}

func NewBlockRequestPool(chain *core.BlockChain, networkManager network.Manager) *blockRequestPool {
	return &blockRequestPool{
		logger: logger.NewLogger("catchup"),
		chain:  chain,

		networkManager:  networkManager,
		peers:           make(map[message.PeerId]*peer),
		requestedBlocks: make(map[uint64]*blockRequest),
	}
}

func (o *blockRequestPool) Start() {
	o.startTime = time.Now()
	go o.syncBlocks()
}

func (o *blockRequestPool) Stop() {
}

func (o *blockRequestPool) OnConnect(id message.PeerId) {
	o.networkManager.Send(id, &message.BlockHeightRequestMessage{o.chain.State().Height})
}

func (o *blockRequestPool) Receive(id message.PeerId, msg interface{}) {
	currentHeight := o.chain.State().Height
	switch msg := msg.(type) {
	case *message.BlockHeightRequestMessage:
		o.logger.Info("msg", "Received blockHeightRequest",
			"from", id.Abbr(6), "our", currentHeight, "height", msg.Height)
		o.setBlockHeight(id, msg.Height)
		o.networkManager.Send(id, &message.BlockHeightResponseMessage{currentHeight})
	case *message.BlockHeightResponseMessage:
		o.logger.Info("msg", "Received blockHeightResponse",
			"from", id.Abbr(6), "our", currentHeight, "height", msg.Height)
		o.setBlockHeight(id, msg.Height)
	case *message.BlockRequestMessage:
		// TODO: block validation & apply block
	}
}

func (o *blockRequestPool) syncBlocks() {
	syncTimer := time.NewTimer(blockSyncInterval)
	defer syncTimer.Stop()

	for {
		select {
		case <-syncTimer.C:
			for o.remainBlocks() {
				o.nextBlock()
			}

			syncTimer.Reset(blockSyncInterval)
		}
	}
}

// Check remain blocks
func (o *blockRequestPool) remainBlocks() bool {
	if o.chain.State() != nil {
		chainState := o.chain.State()
		lastBlockHeight := chainState.Height

		if lastBlockHeight < o.latestHeight {
			lastBlockHeight = o.latestHeight
		}

		delta := o.highestHeight - o.lowestHeight
		return delta < blockSyncMaxConcurrentRequests && o.highestHeight < lastBlockHeight
	} else {
		return false
	}
}

func (o *blockRequestPool) nextBlock() {
	o.lock.Lock()
	defer o.lock.Unlock()

	nextHeight := o.lowestHeight + 1
	if o.highestHeight == 0 {
		nextHeight = o.chain.State().Height + 1
	}

	peer := o.pickPeer()
	o.makeRequest(&blockRequest{
		peerId:    peer.id,
		height:    nextHeight,
		startTime: time.Now(),
	})
}

func (o *blockRequestPool) makeRequest(request *blockRequest) {
	// TODO: block request
	o.networkManager.Send(request.peerId, &message.BlockHeightRequestMessage{})
}

// Load Balance
func (o *blockRequestPool) pickPeer() *peer {
	var lowestPeer *peer
	var lowestPendings int32

	for _, peer := range o.peers {
		if lowestPendings == 0 || lowestPendings > peer.pendings {
			lowestPeer = peer
			lowestPendings = peer.pendings
		}
	}

	return lowestPeer
}

// Set peer's height
func (o *blockRequestPool) setBlockHeight(id message.PeerId, height uint64) {
	o.lock.Lock()
	defer o.lock.Unlock()

	if _, ok := o.peers[id]; !ok {
		o.peers[id] = &peer{
			id:     id,
			height: height,
		}
	} else {
		o.peers[id].height = height
	}

	if o.latestHeight < height {
		o.latestHeight = height
	}
}
