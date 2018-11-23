package sync

import (
	"context"
	"net/http"
	"time"

	"boscoin.io/sebak/lib/ballot"
	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/transaction"
)

type SyncProgress struct {
	StartingBlock uint64 // Block number where sync began
	CurrentBlock  uint64 // Current block number where sync is at
	HighestBlock  uint64 // Highest alleged block number in the chain
}

type SyncController interface {
	SetSyncTargetBlock(ctx context.Context, height uint64, nodeAddressList []string) error
}

type SyncInfo struct {
	Height uint64
	Block  *block.Block
	Txs    []*transaction.Transaction
	Ptx    *ballot.ProposerTransaction

	// Fetching target node addresses, NodeList is  the validators which
	// participated and confirmed the consensus of latest ballot.
	NodeList *NodeList
}

func (s *SyncInfo) NodeAddrs() []string {
	return s.NodeList.NodeAddrs()
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

type NodeInfo struct {
	Node struct {
		State   string `json:"state"`
		Address string `json:"address"`
		Alias   string `json:"alias"`
	} `json:"node"`
	Block struct {
		Height   int    `json:"height"`
		Hash     string `json:"hash"`
		TotalTxs int    `json:"total-txs"`
		TotalOps int    `json:"total-ops"`
	} `json:"block"`
}
