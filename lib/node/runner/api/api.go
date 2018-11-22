package api

import (
	"boscoin.io/sebak/lib/common/observer"
	"boscoin.io/sebak/lib/transaction"
	"boscoin.io/sebak/lib/transaction/operation"
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
	accountMap := make(map[string]struct{})
	for _, tx := range transactions {
		source := tx.Source()
		accountMap[source] = struct{}{}
		txHash := tx.H.Hash

		txEvent := observer.NewSubscribe(observer.NewEvent(observer.ResourceTransaction, observer.ConditionAll, "")).String()
		txEvent += " " + observer.NewSubscribe(observer.NewEvent(observer.ResourceTransaction, observer.ConditionSource, source)).String()
		txEvent += " " + observer.NewSubscribe(observer.NewEvent(observer.ResourceTransaction, observer.ConditionTxHash, txHash)).String()

		for _, op := range tx.B.Operations {
			opHash := fmt.Sprintf("%s-%s", op.MakeHashString(), txHash)

			opEvent := observer.NewSubscribe(observer.NewEvent(observer.ResourceOperation, observer.ConditionAll, "")).String()
			opEvent += " " + observer.NewSubscribe(observer.NewEvent(observer.ResourceOperation, observer.ConditionTxHash, txHash)).String()
			opEvent += " " + observer.NewSubscribe(observer.NewEvent(observer.ResourceOperation, observer.ConditionOpHash, opHash)).String()

			opEvent += " " + observer.NewSubscribe(observer.NewEvent(observer.ResourceOperation, observer.ConditionSource, source)).String()
			opEvent += " " + observer.NewSubscribe(observer.NewEvent(observer.ResourceOperation, observer.ConditionSource, source), observer.NewEvent(observer.ResourceOperation, observer.ConditionType, string(op.H.Type))).String()
			if pop, ok := op.B.(operation.Tagetable); ok {
				target := pop.TargetAddress()
				accountMap[target] = struct{}{}
				txEvent += " " + observer.NewSubscribe(observer.NewEvent(observer.ResourceTransaction, observer.ConditionTarget, target)).String()
				opEvent += " " + observer.NewSubscribe(observer.NewEvent(observer.ResourceOperation, observer.ConditionTarget, target)).String()
				opEvent += " " + observer.NewSubscribe(observer.NewEvent(observer.ResourceOperation, observer.ConditionTarget, target), observer.NewEvent(observer.ResourceOperation, observer.ConditionType, string(op.H.Type))).String()
			}
			opBlock, _ := block.GetBlockOperation(st, opHash)
			go observer.ResourceObserver.Trigger(opEvent, &opBlock)
		}

		txBlock, _ := block.GetBlockTransaction(st, tx.H.Hash)
		go observer.ResourceObserver.Trigger(txEvent, &txBlock)

	}
	for account, _ := range accountMap {
		accEvent := observer.NewSubscribe(observer.NewEvent(observer.ResourceAccount, observer.ConditionAll, "")).String()
		accEvent += " " + observer.NewSubscribe(observer.NewEvent(observer.ResourceAccount, observer.ConditionAddress, account)).String()
		accountBlock, _ := block.GetBlockAccount(st, account)
		go observer.ResourceObserver.Trigger(accEvent, accountBlock)
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
