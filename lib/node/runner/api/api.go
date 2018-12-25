package api

import (
	"encoding/json"
	"fmt"

	"boscoin.io/sebak/lib/block"
	obs "boscoin.io/sebak/lib/common/observer"
	"boscoin.io/sebak/lib/network"
	"boscoin.io/sebak/lib/network/httputils"
	"boscoin.io/sebak/lib/node"
	"boscoin.io/sebak/lib/node/runner/api/resource"
	"boscoin.io/sebak/lib/storage"
	"boscoin.io/sebak/lib/transaction"
	"boscoin.io/sebak/lib/transaction/operation"
)

const APIVersionV1 = "v1"

// API Endpoint patterns
const (
	GetAccountTransactionsHandlerPattern   = "/accounts/{id}/transactions"
	GetAccountHandlerPattern               = "/accounts/{id}"
	GetAccountsHandlerPattern              = "/accounts"
	GetAccountOperationsHandlerPattern     = "/accounts/{id}/operations"
	GetAccountFrozenAccountHandlerPattern  = "/accounts/{id}/frozen-accounts"
	GetFrozenAccountHandlerPattern         = "/frozen-accounts"
	GetTransactionsHandlerPattern          = "/transactions"
	GetTransactionByHashHandlerPattern     = "/transactions/{id}"
	GetTransactionOperationsHandlerPattern = "/transactions/{id}/operations"
	GetTransactionOperationHandlerPattern  = "/transactions/{id}/operations/{opindex}"
	GetTransactionStatusHandlerPattern     = "/transactions/{id}/status"
	PostTransactionPattern                 = "/transactions"
	GetBlocksHandlerPattern                = "/blocks"
	GetBlockHandlerPattern                 = "/blocks/{hashOrHeight}"
	GetNodeInfoPattern                     = "/"
	PostSubscribePattern                   = "/subscribe"
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

func TriggerEvent(st *storage.LevelDBBackend, transactions []*transaction.Transaction) {
	var (
		t     = obs.ResourceObserver.Trigger
		cond  = obs.NewCondition
		event = obs.Event
	)

	accountMap := make(map[string]struct{})

	for _, tx := range transactions {
		source := tx.Source()
		accountMap[source] = struct{}{}
		txHash := tx.H.Hash
		bt, err := block.GetBlockTransaction(st, tx.H.Hash)
		if err != nil {
			return
		}

		t(event(cond(obs.Tx, obs.All)), &bt)
		t(event(cond(obs.Tx, obs.Source, source)), &bt)
		t(event(cond(obs.Tx, obs.TxHash, txHash)), &bt)

		for _, op := range tx.B.Operations {
			if pop, ok := op.B.(operation.Targetable); ok {
				target := pop.TargetAddress()
				accountMap[target] = struct{}{}
				t(event(cond(obs.Tx, obs.Target, target)), &bt)
			}
		}
	}
	for account, _ := range accountMap {
		ba, err := block.GetBlockAccount(st, account)
		if err != nil {
			return
		}
		t(event(cond(obs.Acc, obs.All)), ba)
		t(event(cond(obs.Acc, obs.Address, account)), ba)
	}

}

func renderEventStream(args ...interface{}) ([]byte, error) {
	if len(args) <= 1 {
		return nil, fmt.Errorf("render: value is empty") //TODO(anarcher): Error type
	}
	i := args[1]

	if i == nil {
		return []byte{}, nil
	}

	switch v := i.(type) {
	case *block.BlockAccount:
		r := resource.NewAccount(v)
		return json.Marshal(r.Resource())
	case *block.BlockTransaction:
		r := resource.NewTransaction(v)
		return json.Marshal(r.Resource())
	case httputils.HALResource:
		return json.Marshal(v.Resource())
	}

	return json.Marshal(i)
}
