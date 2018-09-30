package sync

import (
	"context"
	"net/http"
	"time"

	"boscoin.io/sebak/lib/block"
)

type BlockInfo struct {
	BlockHeight uint64
	Block       *block.Block
	Txs         []*block.BlockTransaction
	Ops         []*block.BlockOperation
}

type Doer interface {
	Do(*http.Request) (*http.Response, error)
}

type AfterFunc = func(time.Duration) <-chan time.Time

type Fetcher interface {
	Fetch(ctx context.Context, height uint64) (*BlockInfo, error)
}

type Validator interface {
	Validate(context.Context, *BlockInfo) error
}
