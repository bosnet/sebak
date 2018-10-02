package sync

import (
	"context"
	"net/http"
	"time"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/transaction"
)

type SyncProgress struct {
	StartingBlock uint64 // Block number where sync began
	CurrentBlock  uint64 // Current block number where sync is at
	HighestBlock  uint64 // Highest alleged block number in the chain
}

type SyncController interface {
	SetSyncTargetBlock(ctx context.Context, height uint64) error
}

type SyncInfo struct {
	BlockHeight uint64
	Block       *block.Block
	Txs         []*transaction.Transaction

	BlockTxs []*block.BlockTransaction
	BlockOps []*block.BlockOperation
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
