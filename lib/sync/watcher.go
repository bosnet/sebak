package sync

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"golang.org/x/sync/errgroup"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/errors"
	"boscoin.io/sebak/lib/network"
	"boscoin.io/sebak/lib/node"
	"boscoin.io/sebak/lib/node/runner/api"
	"boscoin.io/sebak/lib/storage"
	"github.com/inconshreveable/log15"
)

type Watcher struct {
	syncer    SyncController
	cm        network.ConnectionManager
	st        *storage.LevelDBBackend
	localNode *node.LocalNode
	client    Doer
	after     AfterFunc
	interval  time.Duration
	stop      chan chan struct{}
	ctx       context.Context
	cancel    context.CancelFunc
	logger    log15.Logger
}

func NewWatcher(
	syncer SyncController,
	client Doer,
	cm network.ConnectionManager,
	st *storage.LevelDBBackend,
	ln *node.LocalNode) *Watcher {
	ctx, cancel := context.WithCancel(context.Background())
	w := &Watcher{
		syncer:    syncer,
		client:    client,
		cm:        cm,
		st:        st,
		localNode: ln,
		after:     time.After,
		interval:  5 * time.Second,
		ctx:       ctx,
		cancel:    cancel,
		logger:    common.NopLogger(),
	}
	return w
}

func (w *Watcher) SetLogger(l log15.Logger) {
	w.logger = l
}

func (w *Watcher) Start() error {
	w.localNode.SetWatch()
	w.loop()
	return nil
}

func (w *Watcher) Stop() error {
	w.cancel()
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
		nodes         []string
		err           error
		ctx           = w.ctx
	)

	latestHeight = w.latestHeight()
	w.logger.Info("starting sync watcher", "height", latestHeight)

L:
	for {
		select {
		case <-checkc:
			highestHeight, nodes, err = w.highestHeightAndNodes(ctx)
			if err != nil {
				if err == context.Canceled {
					break L
				}
				w.logger.Error("get highest height has err", "err", err, "high", highestHeight, "nodes", nodes)
				checkc = w.after(w.interval)
				continue
			}
			if highestHeight > latestHeight {
				w.logger.Info("set sync target block", "high", highestHeight, "nodes", nodes)
				syncer.SetSyncTargetBlock(ctx, highestHeight, nodes)
				latestHeight = highestHeight
			}
			w.logger.Info("watched sync height", "high", highestHeight, "last", latestHeight)
			checkc = w.after(w.interval)
		case c := <-w.stop:
			close(c)
			break L
		}
	}
	w.logger.Info("end sync watcher", "high", highestHeight, "last", latestHeight)

}

func (w *Watcher) highestHeightAndNodes(ctx context.Context) (uint64, []string, error) {
	select {
	case <-ctx.Done():
		return 0, nil, ctx.Err()
	default:
	}

	nodes, err := w.fetchNodeInfos(ctx)
	if err != nil {
		return 0, nil, err
	}

	height, err := w.bestHeightFromNodes(ctx, nodes)
	if err != nil {
		return 0, nil, err
	}

	nodeAddrs, err := w.bestNodeAddrs(ctx, height, nodes)
	if err != nil {
		return 0, nil, err
	}

	w.logger.Info("done get highest height and nodes", "height", height, "nodes", nodeAddrs)

	return height, nodeAddrs, nil
}

func (w *Watcher) fetchNodeInfos(ctx context.Context) ([]*node.NodeInfo, error) {
	var addrs = w.cm.AllConnected()

	w.logger.Debug("start fetch node infos", "addrs", addrs)

	for i, a := range addrs {
		if a == w.localNode.Address() {
			addrs = append(addrs[:i], addrs[i+1:]...)
		}
	}

	if len(addrs) <= 0 {
		return nil, fmt.Errorf("no fetch nodes:")
	}

	var nodes = make([]*node.NodeInfo, len(addrs))
	var g errgroup.Group

	for i, addr := range addrs {
		i, addr := i, addr
		g.Go(func() error {
			node := w.cm.GetNode(addr)
			nodeInfo, err := w.reqNodeInfo(ctx, node)
			if err != nil {
				w.logger.Error("fetch error", "err", err, "node", node)
				return err
			}

			nodes[i] = nodeInfo
			return nil
		})
	}
	g.Wait()

	// it's ok when one of them is alive.
	if len(nodes) <= 0 {
		err := errors.AllValidatorsNotConnected
		w.logger.Error("no one node to alive", "err", err)
		return nil, err
	}

	return nodes, nil
}

func (w *Watcher) bestHeightFromNodes(ctx context.Context, nodes []*node.NodeInfo) (uint64, error) {
	var height uint64

	for _, n := range nodes {
		if n.Node.State != node.StateCONSENSUS {
			w.logger.Info("node state is not CONSENSUS", "node", n.Node.Address)
			continue
		}
		nHeight := uint64(n.Block.Height)
		if nHeight > height {
			height = nHeight
		}
	}

	return height, nil
}

func (w *Watcher) bestNodeAddrs(ctx context.Context, height uint64, nodes []*node.NodeInfo) ([]string, error) {
	var addrs []string
	for _, n := range nodes {
		if n.Node.State != node.StateCONSENSUS {
			continue
		}
		addrs = append(addrs, n.Node.Address)
	}
	return addrs, nil
}

func (w *Watcher) latestHeight() uint64 {
	b := block.GetLatestBlock(w.st)
	return b.Height
}

func (w *Watcher) reqNodeInfo(ctx context.Context, n node.Node) (*node.NodeInfo, error) {
	infoURL := nodeInfoURL(n)
	req, err := http.NewRequest("GET", infoURL.String(), nil)
	if err != nil {
		return nil, err
	}

	req = req.WithContext(ctx)

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	resp, err := w.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		w.logger.Error("node info status code is not 200", "code", resp.StatusCode, "req", infoURL.String())
		return nil, fmt.Errorf("node info response code is not 200")
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var nodeInfo node.NodeInfo
	if err := json.Unmarshal(body, &nodeInfo); err != nil {
		w.logger.Error("resp json error", "err", err, "req", infoURL.String(), "resp", string(body))
		return nil, err
	}
	return &nodeInfo, nil
}

func nodeInfoURL(node node.Node) *url.URL {
	ep := node.Endpoint()
	u := url.URL(*ep)
	u.Path = api.GetNodeInfoPattern
	return &u
}
