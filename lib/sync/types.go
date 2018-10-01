package sync

import (
	"context"
	"net/http"
	"time"

	"boscoin.io/sebak/lib/block"
)

type SyncInfo struct {
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
	Fetch(ctx context.Context, syncInfo *SyncInfo) (*SyncInfo, error)
}

type Validator interface {
	Validate(context.Context, *SyncInfo) error
}
