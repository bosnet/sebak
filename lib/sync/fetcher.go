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
)

type BlockFullFetcher struct {
	network           network.Network
	connectionManager network.ConnectionManager
	apiClient         Doer

	fetchTimeout time.Duration

	reqmsg  <-chan *Message
	reqresp chan *Response

	messages chan *Message
	response <-chan *Response

	stop   chan chan struct{}
	cancel chan chan struct{}
}

type BlockFullFetcherOption = func(f *BlockFullFetcher)

var _ Fetcher = (*BlockFullFetcher)(nil)

func NewBlockFullFetcher(nw network.Network, cManager network.ConnectionManager, opts ...BlockFullFetcherOption) *BlockFullFetcher {
	f := &BlockFullFetcher{
		network:           nw,
		connectionManager: cManager,
		apiClient:         &http.Client{},

		reqmsg:  nil,
		reqresp: make(chan *Response),

		messages: make(chan *Message),
		response: make(chan *Response),

		stop:   make(chan chan struct{}),
		cancel: make(chan chan struct{}),
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
			f.fetch(msg)
		case resp := <-f.response:
			if resp.Err() != nil {
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
	//TODO: fetch block using block node api
	bh := msg.BlockHeight
	n := f.pickRandomNode()
	if n == nil {
		f.errorResponse(msg, errors.New("node not found "))
		return
	}
	ep := n.Endpoint()
	apiURL := url.URL(*ep)
	apiURL.Path = runner.GetBlocksPattern
	apiURL.Query().Set("height-range", fmt.Sprintf("%d-%d", bh, bh+1))
	apiURL.Query().Set("mode", "full")

	req, err := http.NewRequest("GET", apiURL.String(), nil)
	if err != nil {
		f.errorResponse(msg, err)
		return
	}
	ctx, cancelF := context.WithTimeout(context.Background(), f.fetchTimeout)
	defer cancelF()

	req = req.WithContext(ctx)

	resp, err := f.apiClient.Do(req)
	defer resp.Body.Close()
	if err != nil {
		f.errorResponse(msg, err)
		return
	}
	if resp.StatusCode == http.StatusNotFound {
		//TODO:
		err := fmt.Errorf("not found")
		f.errorResponse(msg, err)
		return
	}

	items, err := f.unmarshalResp(resp.Body)
	if err != nil {
		f.errorResponse(msg, err)
		return
	}

	blocks, ok := items[runner.NodeItemBlock]
	if !ok || len(blocks) <= 0 {
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
	idx := rand.Intn(len(ac))
	node := f.connectionManager.GetNode(ac[idx])
	return node
}

func (f *BlockFullFetcher) errorResponse(msg *Message, err error) {
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
