package api

import (
	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/network"
	"boscoin.io/sebak/lib/network/api/resource"
	"boscoin.io/sebak/lib/network/httputils"
	"boscoin.io/sebak/lib/node"
	"boscoin.io/sebak/lib/storage"
	"encoding/json"
	"fmt"
)

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
}

func NewNetworkHandlerAPI(localNode *node.LocalNode, network network.Network, storage *storage.LevelDBBackend) *NetworkHandlerAPI {
	return &NetworkHandlerAPI{
		localNode: localNode,
		network:   network,
		storage:   storage,
	}
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
