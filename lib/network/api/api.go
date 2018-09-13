package api

import (
	"fmt"

	"boscoin.io/sebak/lib/network"
	"boscoin.io/sebak/lib/node"
	"boscoin.io/sebak/lib/storage"
)

const APIVersionV1 = "v1"

// API Endpoint patterns
const (
	GetAccountTransactionsHandlerPattern = "/account/{id}/transactions"
	GetAccountHandlerPattern             = "/account/{id}"
	GetAccountOperationsHandlerPattern   = "/account/{id}/operations"
	GetTransactionsHandlerPattern        = "/transactions"
	GetTransactionByHashHandlerPattern   = "/transactions/{id}"
	PostTransactionPattern               = "/transactions"
)

type NetworkHandlerAPI struct {
	localNode *node.LocalNode
	network   network.Network
	storage   *storage.LevelDBBackend
	urlPrefix string
	version   string
}

func NewNetworkHandlerAPI(localNode *node.LocalNode, network network.Network, storage *storage.LevelDBBackend, urlPrefix string) *NetworkHandlerAPI {
	return &NetworkHandlerAPI{
		localNode: localNode,
		network:   network,
		storage:   storage,
		urlPrefix: urlPrefix,
		version:   APIVersionV1,
	}
}

func (api NetworkHandlerAPI) HandlerURLPattern(pattern string) string {
	return fmt.Sprintf("%s/%s%s", api.urlPrefix, api.version, pattern)
}
