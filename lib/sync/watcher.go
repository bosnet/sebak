package sync

import (
	"context"
	"fmt"
	"time"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/network"
	"boscoin.io/sebak/lib/node"
	"boscoin.io/sebak/lib/storage"
	"github.com/inconshreveable/log15"
)

type Watcher struct {
	cm       network.ConnectionManager
	st       *storage.LevelDBBackend
	syncer   SyncController
	after    AfterFunc
	interval time.Duration
	stop     chan chan struct{}
	logger   log15.Logger
}

func NewWatcher(cm network.ConnectionManager, st *storage.LevelDBBackend, syncer SyncController) *Watcher {
	w := &Watcher{
		cm:       cm,
		st:       st,
		syncer:   syncer,
		after:    time.After,
		interval: 5 * time.Second,
		stop:     make(chan chan struct{}),
		logger:   NopLogger(),
	}
	return w
}

func (w *Watcher) Start() error {
	w.loop()
	return nil
}

func (w *Watcher) Stop() error {
	c := make(chan struct{})
	w.stop <- c
	<-c
	return nil
}

func (w *Watcher) loop() {
	var (
		syncer        = w.syncer
		checkc        = w.after(w.interval)
		highestHeight uint64
		latestHeight  uint64
		err           error
		ctx           = context.Background()
	)
	latestHeight = w.latestHeight()
	w.logger.Info("starting sync watcher", "height", latestHeight)

	for {
		select {
		case <-checkc:
			highestHeight, err = w.highestHeight()
			if err != nil {
				w.logger.Error(err.Error(), "err", err.Error())
				continue
			}
			if highestHeight > latestHeight {
				syncer.SetSyncTargetBlock(ctx, highestHeight)
				latestHeight = highestHeight
			}
			w.logger.Info("watched sync height", "high", highestHeight)
			checkc = w.after(w.interval)
		case c := <-w.stop:
			close(c)
			return
		}
	}
}

func (w *Watcher) highestHeight() (uint64, error) {
	var (
		ac            = w.cm.AllConnected()
		nodes         []node.Node
		highestHeight uint64
	)

	for _, a := range ac {
		n := w.cm.GetNode(a)
		if n == nil {
			continue
		}
		nodes = append(nodes, n)
		if n.BlockHeight() > highestHeight {
			highestHeight = n.BlockHeight()
			w.logger.Info(fmt.Sprintf("node %v has highestHeight %v", n, highestHeight), "height", highestHeight)
		}
	}

	return highestHeight, nil
}

func (w *Watcher) latestHeight() uint64 {
	blk, err := block.GetLatestBlock(w.st)
	if err != nil {
		w.logger.Error("block.GetLatestBlock", "err", err)
		return 0
	}
	return blk.Height
}
