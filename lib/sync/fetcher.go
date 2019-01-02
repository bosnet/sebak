package sync

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"time"

	"boscoin.io/sebak/lib/ballot"
	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/errors"
	"boscoin.io/sebak/lib/metrics"
	"boscoin.io/sebak/lib/network"
	"boscoin.io/sebak/lib/node"
	"boscoin.io/sebak/lib/node/runner"
	api "boscoin.io/sebak/lib/node/runner/node_api"
	"boscoin.io/sebak/lib/storage"

	"github.com/inconshreveable/log15"
)

type BlockFetcher struct {
	connectionManager network.ConnectionManager
	apiClient         Doer
	storage           *storage.LevelDBBackend
	localNode         *node.LocalNode

	fetchTimeout  time.Duration
	retryInterval time.Duration

	logger log15.Logger
}

type BlockFetcherOption = func(f *BlockFetcher)

func NewBlockFetcher(
	cm network.ConnectionManager,
	client Doer,
	st *storage.LevelDBBackend,
	localNode *node.LocalNode,
	opts ...BlockFetcherOption) *BlockFetcher {

	f := &BlockFetcher{
		connectionManager: cm,
		apiClient:         client,
		storage:           st,
		localNode:         localNode,
		logger:            common.NopLogger(),

		fetchTimeout:  1 * time.Minute,
		retryInterval: 30 * time.Second,
	}

	for _, opt := range opts {
		opt(f)
	}

	return f
}

func (f *BlockFetcher) Fetch(ctx context.Context, syncInfo *SyncInfo) (*SyncInfo, error) {
	height := syncInfo.Height

	TryForever(func(attempt int) (bool, error) {
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		default:
			f.logger.Debug("try to fetch", "height", height, "attempt", attempt)
			if err := f.fetch(ctx, syncInfo); err != nil {
				if err == context.Canceled {
					return false, ctx.Err()
				}
				f.logger.Error("fetch err", "err", err, "height", height)
				metrics.Sync.AddFetchError()
				c := time.After(f.retryInterval) //afterFunc?
				select {
				case <-ctx.Done():
					return false, ctx.Err()
				case <-c:
					return true, err
				}
			}
			return false, nil
		}
	})

	return syncInfo, nil
}

func (f *BlockFetcher) fetch(ctx context.Context, si *SyncInfo) error {
	var (
		height    = si.Height
		nodeAddrs = si.NodeAddrs()
	)
	si.Bts = si.Bts[:0]
	f.logger.Debug("start fetch", "height", height, "nodes", nodeAddrs)

	if len(nodeAddrs) <= 0 {
		f.logger.Error("Node addrs are nil!", "height", height, "nodes", nodeAddrs)
		return errors.NodeNotFound
	}

	n := f.pickRandomNode(nodeAddrs)
	if n == nil {
		f.logger.Error("Alive Node addrs not exists", "height", height, "nodes", nodeAddrs)
		return errors.NodeNotFound
	}
	f.logger.Debug("fetching items from node", "fetching_node", n, "height", height)

	apiURL := apiClientURL(n, height)
	f.logger.Debug("apiClient", "url", apiURL.String())

	req, err := http.NewRequest("GET", apiURL.String(), nil)
	if err != nil {
		err := errors.Wrap(err, "api request")
		f.logger.Error("request err", "err", err, "height", height)
		return err
	}
	req = req.WithContext(ctx)

	resp, err := f.apiClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return errors.New("fetch: block not found")
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		return errors.New("fetch: too many requests")
	}

	items, err := f.unmarshalResp(resp.Body)
	if err != nil {
		err := errors.Wrap(err, "response failed to unmarshal")
		code := resp.StatusCode
		f.logger.Debug("unmarshalResp err", "err", err, "height", height, "statusCode", code)
		return err
	}

	f.logger.Debug("fetch get items", "items", len(items), "height", height)

	blocks, ok := items[api.NodeItemBlock]
	if !ok || len(blocks) <= 0 {
		err := errors.New("fetch: block not found in response")
		return err
	}

	bts, ok := items[api.NodeItemBlockTransaction]
	if !ok {
		err := errors.New("fetch: block transactions not found in response")
		return err
	}

	blk := blocks[0].(block.Block)
	si.Block = &blk

	{
		btmap := make(map[string]*block.BlockTransaction) // For ordering txs by block.Transactions

		for _, bt := range bts {
			bt, ok := bt.(block.BlockTransaction)
			if !ok {
				return errors.InvalidTransaction
			}
			btmap[bt.Hash] = &bt
		}

		for _, hash := range blk.Transactions {
			bt, ok := btmap[hash]
			if !ok {
				return errors.Wrapf(errors.TransactionNotFound, "block hash: %s height: %d", hash, blk.Height)
			}
			if bt.Transaction().IsEmpty() {
				return errors.Wrapf(errors.TransactionNotFound, "tx in btx not found: tx %s not found of height %s", hash, blk.Height)
			}
			si.Bts = append(si.Bts, bt)
		}

		if blk.ProposerTransaction != "" {
			if bt, ok := btmap[blk.ProposerTransaction]; ok {
				if bt.Transaction().IsEmpty() {
					return errors.Wrapf(errors.TransactionNotFound, "proposer tx in btx not found: tx %s not found of height %s", blk.ProposerTransaction, blk.Height)
				}
				ptx := &ballot.ProposerTransaction{Transaction: bt.Transaction()}
				si.Ptx = ptx
			} else {
				return errors.Wrapf(errors.TransactionNotFound, "proposer transaction block hash: %v", blk.ProposerTransaction)
			}
		}
	}

	f.logger.Debug("end fetch", "height", height)
	return nil
}

// pickRandomNode choose one node by random. It is very protype for choosing fetching which node
func (f *BlockFetcher) pickRandomNode(nodeAddrs []string) node.Node {
	ac := f.connectionManager.AllConnected()
	if len(ac) <= 0 {
		return nil
	}

	var nodeMap = make(map[string]struct{})
	for _, addr := range nodeAddrs {
		nodeMap[addr] = struct{}{}
	}

	var addressList []string
	for _, a := range ac {
		if f.localNode.Address() == a {
			continue
		}
		if len(nodeAddrs) > 0 {
			if _, ok := nodeMap[a]; ok {
				addressList = append(addressList, a)
			}
		} else {
			addressList = append(addressList, a)
		}
	}

	if len(addressList) <= 0 {
		return nil
	}

	idx := rand.Intn(len(addressList))
	node := f.localNode.Validator(addressList[idx])
	return node
}

func (f *BlockFetcher) existsBlockHeight(height uint64) bool {
	exists, err := block.ExistsBlockByHeight(f.storage, height)
	if err != nil {
		f.logger.Error("block.ExistsBlockByHeight", "err", err)
		return false
	}
	return exists
}

func (f *BlockFetcher) unmarshalResp(body io.Reader) (map[api.NodeItemDataType][]interface{}, error) {
	items := map[api.NodeItemDataType][]interface{}{}

	r := bufio.NewReader(body)
	var (
		line []byte
		err  error
	)
	for {
		line, err = r.ReadBytes('\n')
		if err == io.EOF {
			break
		}
		if len(line) <= 0 {
			continue
		}
		itemType, b, err := api.UnmarshalNodeItemResponse(line)
		if err != nil {
			return nil, err
		}
		items[itemType] = append(items[itemType], b)
	}

	if err != nil && err != io.EOF {
		return nil, err
	}

	return items, nil
}

func apiClientURL(n node.Node, height uint64) *url.URL {
	ep := n.Endpoint()
	u := url.URL(*ep)
	u.Path = network.UrlPathPrefixNode + runner.GetBlocksPattern
	q := u.Query()
	q.Set("height-range", fmt.Sprintf("%d-%d", height, height+1))
	q.Set("mode", "full")
	u.RawQuery = q.Encode()

	return &u
}
