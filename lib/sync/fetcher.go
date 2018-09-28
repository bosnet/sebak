package sync

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"time"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/network"
	"boscoin.io/sebak/lib/node"
	"boscoin.io/sebak/lib/node/runner"
	"boscoin.io/sebak/lib/storage"

	"github.com/inconshreveable/log15"
)

type BlockFullFetcher struct {
	network           network.Network
	connectionManager network.ConnectionManager
	apiClient         Doer
	storage           *storage.LevelDBBackend

	fetchTimeout time.Duration

	reqmsg  <-chan *Message
	reqresp chan *Response

	messages chan *Message
	response <-chan *Response

	stop   chan chan struct{}
	cancel chan chan struct{}

	logger log15.Logger
}

type BlockFullFetcherOption = func(f *BlockFullFetcher)

var _ Fetcher = (*BlockFullFetcher)(nil)

func NewBlockFullFetcher(nw network.Network, cManager network.ConnectionManager, st *storage.LevelDBBackend, opts ...BlockFullFetcherOption) *BlockFullFetcher {
	f := &BlockFullFetcher{
		network:           nw,
		connectionManager: cManager,
		apiClient:         &http.Client{},
		storage:           st,

		reqmsg:  nil,
		reqresp: make(chan *Response),

		messages: make(chan *Message),
		response: make(chan *Response),

		stop:   make(chan chan struct{}),
		cancel: make(chan chan struct{}),

		logger: NopLogger(),
	}

	for _, o := range opts {
		o(f)
	}
	return f
}

// Stopper

func (f *BlockFullFetcher) Stop() error {
	c := make(chan struct{})
	f.stop <- c
	<-c

	return nil
}

// Producer

func (f *BlockFullFetcher) SetResponse(res <-chan *Response) error {
	f.response = res
	return nil
}

func (f *BlockFullFetcher) Produce() <-chan *Message {
	return f.messages
}

// Consumer

func (f *BlockFullFetcher) Consume(msg <-chan *Message) error {
	f.reqmsg = msg
	go f.loop()
	return nil
}

func (f *BlockFullFetcher) Response() <-chan *Response {
	return f.reqresp
}

func (f *BlockFullFetcher) loop() {
	for {
		select {
		case msg := <-f.reqmsg:
			f.logger.Info("Receive message ", "height", msg.BlockHeight)
			exists := f.existsBlockHeight(msg.BlockHeight)
			if exists {
				f.logger.Info("Block already exists", "height", msg.BlockHeight)
				continue
			}
			f.fetch(msg)
		case resp := <-f.response:
			if resp.Err() != nil {
				f.logger.Error("Receive Response", "err", resp.Err(), "height", resp.Message().BlockHeight) //TODO(anarcher): resp
				f.fetch(resp.Message())
			}
		case c := <-f.cancel:
			close(c)
			return
		case c := <-f.stop:
			close(c)
			return

		}
	}
}

func (f *BlockFullFetcher) fetch(msg *Message) {
	f.logger.Debug("Fetch start", "height", msg.BlockHeight)
	//TODO: fetch block using block node api
	bh := msg.BlockHeight
	n := f.pickRandomNode()
	f.logger.Info("Try to fetch from", "node", n, "height", msg.BlockHeight)
	if n == nil {
		f.errorResponse(msg, errors.New("node not found "))
		return
	}
	ep := n.Endpoint()
	apiURL := url.URL(*ep)
	apiURL.Path = network.UrlPathPrefixNode + runner.GetBlocksPattern
	q := apiURL.Query()
	q.Set("height-range", fmt.Sprintf("%d-%d", bh, bh+1))
	q.Set("mode", "full")
	apiURL.RawQuery = q.Encode()
	f.logger.Debug("apiClient", "url", apiURL.String())

	req, err := http.NewRequest("GET", apiURL.String(), nil)
	if err != nil {
		f.errorResponse(msg, err)
		return
	}
	ctx, cancelF := context.WithTimeout(context.Background(), f.fetchTimeout)
	defer cancelF()

	req = req.WithContext(ctx)

	resp, err := f.apiClient.Do(req)
	defer func() {
		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}
	}()
	if err != nil {
		f.errorResponse(msg, err)
		return
	}
	if resp.StatusCode == http.StatusNotFound {
		//TODO:
		err := errors.New("block not found")
		f.errorResponse(msg, err)
		return
	}

	items, err := f.unmarshalResp(resp.Body)
	if err != nil {
		f.errorResponse(msg, err)
		return
	}

	f.logger.Info("Get items", "items", len(items), "height", msg.BlockHeight)

	blocks, ok := items[runner.NodeItemBlock]
	if !ok || len(blocks) <= 0 {
		err := errors.New("block not found in resp")
		f.errorResponse(msg, err)
		return
	}
	//TODO(anarcher): check items
	txs, ok := items[runner.NodeItemBlockTransaction]
	ops, ok := items[runner.NodeItemBlockOperation]

	blk := blocks[0].(block.Block)
	msg.Block = &blk

	for _, tx := range txs {
		bt := tx.(block.BlockTransaction)
		msg.Txs = append(msg.Txs, &bt)
	}
	for _, op := range ops {
		op := op.(block.BlockOperation)
		msg.Ops = append(msg.Ops, &op)
	}

	select {
	case f.messages <- msg:
	case c := <-f.stop:
		f.cancel <- c
	}
	f.logger.Debug("Fetched", "height", msg.BlockHeight)
}

func (f *BlockFullFetcher) unmarshalResp(body io.ReadCloser) (map[runner.NodeItemDataType][]interface{}, error) {
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

// pickRandomNode choose one node by random. It is very protype for choosing fetching which node
func (f *BlockFullFetcher) pickRandomNode() node.Node {
	ac := f.connectionManager.AllConnected()
	if len(ac) <= 0 {
		return nil
	}
	idx := rand.Intn(len(ac))
	node := f.connectionManager.GetNode(ac[idx])
	return node
}

func (f *BlockFullFetcher) errorResponse(msg *Message, err error) {
	f.logger.Error("Error response", "err", err, "height", msg.BlockHeight)
	resp := &Response{
		err: err,
		msg: msg,
	}
	select {
	case f.reqresp <- resp:
	case c := <-f.stop:
		f.cancel <- c
	}
}

func (f *BlockFullFetcher) existsBlockHeight(height uint64) bool {
	exists, err := block.ExistsBlockByHeight(f.storage, height)
	if err != nil {
		f.logger.Error("block.ExistsBlockByHeight", "err", err)
		return false
	}
	return exists
}
