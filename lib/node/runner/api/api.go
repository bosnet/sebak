package api

import (
	"encoding/json"
	"fmt"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/network"
	"boscoin.io/sebak/lib/network/httputils"
	"boscoin.io/sebak/lib/node"
	"boscoin.io/sebak/lib/node/runner/api/resource"
	"boscoin.io/sebak/lib/storage"
)

const APIVersionV1 = "v1"

// API Endpoint patterns
const (
	GetAccountTransactionsHandlerPattern   = "/accounts/{id}/transactions"
	GetAccountHandlerPattern               = "/accounts/{id}"
	GetAccountOperationsHandlerPattern     = "/accounts/{id}/operations"
	GetAccountFrozenAccountHandlerPattern  = "/accounts/{id}/frozen-accounts"
	GetFrozenAccountHandlerPattern         = "/frozen-accounts"
	GetTransactionsHandlerPattern          = "/transactions"
	GetTransactionByHashHandlerPattern     = "/transactions/{id}"
	GetTransactionOperationsHandlerPattern = "/transactions/{id}/operations"
	PostTransactionPattern                 = "/transactions"
	GetTransactionHistoryHandlerPattern    = "/transactions/{id}/history"
	GetBlocksHandlerPattern                = "/blocks"
	GetBlockHandlerPattern                 = "/blocks/{hashOrHeight}"
	GetNodeInfoPattern                     = "/"
)

type NetworkHandlerAPI struct {
	localNode      *node.LocalNode
	network        network.Network
	storage        *storage.LevelDBBackend
	urlPrefix      string
	version        string
	nodeInfo       node.NodeInfo
	GetLatestBlock func() block.Block
}

func NewNetworkHandlerAPI(localNode *node.LocalNode, network network.Network, storage *storage.LevelDBBackend, urlPrefix string, nodeInfo node.NodeInfo) *NetworkHandlerAPI {
	return &NetworkHandlerAPI{
		localNode: localNode,
		network:   network,
		storage:   storage,
		urlPrefix: urlPrefix,
		version:   APIVersionV1,
		nodeInfo:  nodeInfo,
	}
}

func (api NetworkHandlerAPI) HandlerURLPattern(pattern string) string {
	return fmt.Sprintf("%s/%s%s", api.urlPrefix, api.version, pattern)
}

func renderEventStream(args ...interface{}) ([]byte, error) {
	if len(args) <= 1 {
		return nil, fmt.Errorf("render: value is empty") //TODO(anarcher): Error type
	}
	i := args[1]

	if i == nil {
		return nil, nil
	}

	switch v := i.(type) {
	case *block.BlockAccount:
		r := resource.NewAccount(v)
		return json.Marshal(r.Resource())
	case *block.BlockOperation:
		r := resource.NewOperation(v)
		return json.Marshal(r.Resource())
	case *block.BlockTransaction:
		r := resource.NewTransaction(v)
		return json.Marshal(r.Resource())
	case httputils.HALResource:
		return json.Marshal(v.Resource())
	}

	return json.Marshal(i)
}
