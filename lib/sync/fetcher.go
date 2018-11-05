package sync

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"time"

	"boscoin.io/sebak/lib/ballot"
	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/errors"
	"boscoin.io/sebak/lib/network"
	"boscoin.io/sebak/lib/node"
	"boscoin.io/sebak/lib/node/runner"
	"boscoin.io/sebak/lib/storage"
	"boscoin.io/sebak/lib/transaction"

	"github.com/inconshreveable/log15"
)

type BlockFetcher struct {
	network           network.Network
	connectionManager network.ConnectionManager
	apiClient         Doer
	storage           *storage.LevelDBBackend
	localNode         *node.LocalNode

	fetchTimeout  time.Duration
	retryInterval time.Duration

	logger log15.Logger
}

type BlockFetcherOption = func(f *BlockFetcher)

func NewBlockFetcher(nw network.Network,
	cManager network.ConnectionManager,
	st *storage.LevelDBBackend,
	localNode *node.LocalNode,
	opts ...BlockFetcherOption) *BlockFetcher {

	f := &BlockFetcher{
		network:           nw,
		connectionManager: cManager,
		storage:           st,
		localNode:         localNode,
		logger:            common.NopLogger(),

		fetchTimeout:  1 * time.Minute,
		retryInterval: 30 * time.Second,
	}

	for _, opt := range opts {
		opt(f)
	}

	client, err := common.NewHTTP2Client(f.fetchTimeout, 0, true)
	if err != nil {
		f.logger.Error("make http2 client", "err", err)
		panic(err) // It's an unrecoverable error not to make client when starting syncer / node
	}
	f.apiClient = client

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
		nodeAddrs = si.NodeAddrs
	)
	f.logger.Debug("fetch start", "height", height)

	n := f.pickRandomNode(nodeAddrs)
	if n == nil {
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

	resp, err := f.apiClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return errors.New("fetch: block not found")
	}

	items, err := f.unmarshalResp(resp.Body)
	if err != nil {
		err := errors.Wrap(err, "reponse failed to unmarshal")

		body := func() string {
			body, readErr := ioutil.ReadAll(resp.Body)
			if readErr != nil {
				return readErr.Error()
			}
			return string(body)
		}
		f.logger.Debug("unmarshalResp err", "err", err, "height", height, "statusCode", resp.StatusCode, "body", log15.Lazy{Fn: body})
		return err
	}

	f.logger.Debug("fetch get items", "items", len(items), "height", height)

	blocks, ok := items[runner.NodeItemBlock]
	if !ok || len(blocks) <= 0 {
		err := errors.New("fetch: block not found in response")
		return err
	}

	bts, ok := items[runner.NodeItemBlockTransaction]
	if !ok {
		err := errors.New("fetch: block transactions not found in response")
		return err
	}

	blk := blocks[0].(block.Block)
	si.Block = &blk

	txmap := make(map[string]*transaction.Transaction) // For ordering txs by block.Transactions

	for _, bt := range bts {
		bt, ok := bt.(block.BlockTransaction)
		if !ok {
			return errors.InvalidTransaction
		}

		var tx transaction.Transaction
		if err := json.Unmarshal(bt.Message, &tx); err != nil {
			err := errors.Wrap(err, "transaction.Message unmarshaling failed")
			f.logger.Error("tx.Message unmarshal err", "err", err, "height", height, "message", string(bt.Message), "statusCode", resp.StatusCode)
			return err
		}
		txmap[bt.Hash] = &tx
	}

	for _, hash := range blk.Transactions {
		tx, ok := txmap[hash]
		if !ok {
			return errors.Wrapf(errors.TransactionNotFound, "block hash: %s height: %d", hash, height)
		}
		si.Txs = append(si.Txs, tx)
	}

	if blk.ProposerTransaction != "" {
		if tx, ok := txmap[blk.ProposerTransaction]; ok {
			ptx := &ballot.ProposerTransaction{Transaction: *tx}
			si.Ptx = ptx
		} else {
			return errors.Wrapf(errors.TransactionNotFound, "proposer transaction block hash: %v", blk.ProposerTransaction)
		}
	}

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
	node := f.connectionManager.GetNode(addressList[idx])
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

func (f *BlockFetcher) unmarshalResp(body io.ReadCloser) (map[runner.NodeItemDataType][]interface{}, error) {
	items := map[runner.NodeItemDataType][]interface{}{}

	sc := bufio.NewScanner(body)
	for sc.Scan() {
		itemType, b, err := runner.UnmarshalNodeItemResponse(sc.Bytes())
		if err != nil {
			return nil, err
		}
		items[itemType] = append(items[itemType], b)
	}
	if err := sc.Err(); err != nil {
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
