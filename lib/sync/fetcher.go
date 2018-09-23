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

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/network"
	"boscoin.io/sebak/lib/node/runner"
)

type BlockFullFetcher struct {
	network           network.Network
	connectionManager *network.ConnectionManager
	apiClient         Doer

	fetchTimeout time.Duration

	reqmsg  <-chan *Message
	reqresp chan *Response

	messages chan *Message
	response <-chan *Response

	stop chan chan struct{}
}

type BlockFullFetcherOption = func(f *BlockFullFetcher)

var _ Fetcher = (*BlockFullFetcher)(nil)

func NewBlockFullFetcher(nw network.Network, cManager *network.ConnectionManager, opts ...BlockFullFetcherOption) *BlockFullFetcher {
	f := &BlockFullFetcher{
		network:           nw,
		connectionManager: cManager,
		apiClient:         &http.Client{},

		reqmsg:  nil,
		reqresp: make(chan *Response),

		messages: make(chan *Message),
		response: make(chan *Response),

		stop: make(chan chan struct{}),
	}

	for _, o := range opts {
		o(f)
	}
	go f.loop()
	return f
}

// Stopper

func (f *BlockFullFetcher) Stop() error {

	return nil
}

// Producer

func (f *BlockFullFetcher) Produce(res <-chan *Response) {
	f.response = res
}

func (f *BlockFullFetcher) Message() <-chan *Message {
	return f.messages
}

// Consumer

func (f *BlockFullFetcher) Consume(msg <-chan *Message) error {
	f.reqmsg = msg
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
		case c := <-f.stop:
			close(c)
			return

		}
	}
}

func (f *BlockFullFetcher) fetch(msg *Message) {
	//TODO: fetch block using block node api
	bh := msg.BlockHeight
	nclient := f.pickRandomNode()
	apiURL, err := url.Parse(nclient.Endpoint().String())
	if err != nil {
		f.errorResponse(msg, err)
		return
	}
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

	blk, ok := items[runner.GetBlocksDataTypeBlock]
	if !ok || len(blk) <= 0 {
		f.errorResponse(msg, err)
		return
	}
	txs, ok := items[runner.GetBlocksDataTypeTransaction]
	if !ok {
		f.errorResponse(msg, err)
		return
	}
	ops, ok := items[runner.GetBlocksDataTypeOperation]
	if !ok {
		f.errorResponse(msg, err)
		return
	}

	msg.Block = blk[0].(*block.Block)
	for _, tx := range txs {
		msg.Txs = append(msg.Txs, tx.(*block.BlockTransaction))
	}
	for _, op := range ops {
		msg.Ops = append(msg.Ops, op.(*block.BlockOperation))
	}

	f.messages <- msg
}

func (f *BlockFullFetcher) unmarshalResp(body io.ReadCloser) (map[runner.GetBlocksDataType][]interface{}, error) {
	items := map[runner.GetBlocksDataType][]interface{}{}

	sc := bufio.NewScanner(body)
	for sc.Scan() {
		itemType, b, err := runner.UnmarshalGetBlocksHandlerItem(sc.Bytes())
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
func (f *BlockFullFetcher) pickRandomNode() network.NetworkClient {
	ac := f.connectionManager.AllConnected()
	idx := rand.Intn(len(ac))
	return f.connectionManager.GetConnection(ac[idx])
}

func (f *BlockFullFetcher) errorResponse(msg *Message, err error) {
	f.reqresp <- &Response{
		err: err,
		msg: msg,
	}
}
