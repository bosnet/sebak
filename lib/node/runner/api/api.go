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

		txEvent := observer.NewCondition(observer.ResourceTransaction, observer.KeyAll, "").Event()
		txEvent += " " + observer.NewCondition(observer.ResourceTransaction, observer.KeySource, source).Event()
		txEvent += " " + observer.NewCondition(observer.ResourceTransaction, observer.KeyTxHash, txHash).Event()

		for _, op := range tx.B.Operations {
			opHash := fmt.Sprintf("%s-%s", op.MakeHashString(), txHash)

			opEvent := observer.NewCondition(observer.ResourceOperation, observer.KeyAll, "").Event()
			opEvent += " " + observer.NewCondition(observer.ResourceOperation, observer.KeyTxHash, txHash).Event()
			opEvent += " " + observer.NewCondition(observer.ResourceOperation, observer.KeyOpHash, opHash).Event()

			opEvent += " " + observer.NewCondition(observer.ResourceOperation, observer.KeySource, source).Event()
			opEvent += " " + observer.Conditions{observer.NewCondition(observer.ResourceOperation, observer.KeySource, source), observer.NewCondition(observer.ResourceOperation, observer.KeyType, string(op.H.Type))}.Event()
			if pop, ok := op.B.(operation.Targetable); ok {
				target := pop.TargetAddress()
				accountMap[target] = struct{}{}
				txEvent += " " + observer.NewCondition(observer.ResourceTransaction, observer.KeyTarget, target).Event()
				opEvent += " " + observer.NewCondition(observer.ResourceOperation, observer.KeyTarget, target).Event()
				opEvent += " " + observer.Conditions{observer.NewCondition(observer.ResourceOperation, observer.KeyTarget, target), observer.NewCondition(observer.ResourceOperation, observer.KeyType, string(op.H.Type))}.Event()
			}
			opBlock, _ := block.GetBlockOperation(st, opHash)
			go observer.ResourceObserver.Trigger(opEvent, &opBlock)
		}

		txBlock, _ := block.GetBlockTransaction(st, tx.H.Hash)
		go observer.ResourceObserver.Trigger(txEvent, &txBlock)

	}
	for account, _ := range accountMap {
		accEvent := observer.NewCondition(observer.ResourceAccount, observer.KeyAll, "").Event()
		accEvent += " " + observer.NewCondition(observer.ResourceAccount, observer.KeyAddress, account).Event()
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
