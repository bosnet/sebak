package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/common"
	o "boscoin.io/sebak/lib/common/observer"
	"boscoin.io/sebak/lib/errors"
	"boscoin.io/sebak/lib/network/httputils"
	"boscoin.io/sebak/lib/node/runner/api/resource"
)

func (api NetworkHandlerAPI) GetTransactionsHandler(w http.ResponseWriter, r *http.Request) {
	p, err := NewPageQuery(r)
	if err != nil {
		httputils.WriteJSONError(w, err)
		return
	}

	options, err := p.PageCursorListOptions(common.BlockTransactionPrefixAll)
	if err != nil {
		//TODO: more correct err for it
		httputils.WriteJSONError(w, err)
		return
	}

	var (
		prevOrder *block.BlockOrder
		nextOrder *block.BlockOrder
	)

	readFunc := func() []resource.Resource {
		var txs []resource.Resource
		iterFunc, closeFunc := block.GetBlockTransactions(api.storage, options)
		for {
			t, hasNext, c := iterFunc()
			if !hasNext {
				break
			}
			if prevOrder == nil {
				prevOrder = t.BlockOrder()
			}
			nextOrder = t.BlockOrder()
			txs = append(txs, resource.NewTransaction(&t))
		}
		closeFunc()
		return txs
	}

	txs := readFunc()
	list := p.ResourceListWithOrder(txs, prevOrder, nextOrder)
	httputils.MustWriteJSON(w, 200, list)
}

func (api NetworkHandlerAPI) GetTransactionByHashHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	key := vars["id"]

	readFunc := func() (payload interface{}, err error) {
		found, err := block.ExistsBlockTransaction(api.storage, key)
		if err != nil {
			return nil, err
		}
		if !found {
			return nil, errors.BlockTransactionDoesNotExists
		}
		bt, err := block.GetBlockTransaction(api.storage, key)
		if err != nil {
			return nil, err
		}
		payload = resource.NewTransaction(&bt)
		return payload, nil
	}

	payload, err := readFunc()
	if err == nil {
		httputils.MustWriteJSON(w, 200, payload)
	} else {
		httputils.WriteJSONError(w, err)
	}
}

func (api NetworkHandlerAPI) GetTransactionsByAccountHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	address := vars["id"]

	p, err := NewPageQuery(r)
	if err != nil {
		httputils.WriteJSONError(w, err)
		return
	}

	options, err := p.PageCursorListOptions(block.GetBlockTransactionKeyPrefixAccount(address))
	if err != nil {
		httputils.WriteJSONError(w, err)
		return
	}
	var (
		pOrder *block.BlockOrder
		nOrder
	)
	readFunc := func() []resource.Resource {
		var txs []resource.Resource
		iterFunc, closeFunc := block.GetBlockTransactionsByAccount(api.storage, address, options)
		for {
			t, hasNext, _ := iterFunc()
			if t.BlockOrder() != nil {
				order = t.BlockOrder()
			}
			if !hasNext {
				break
			}
			if pOrder == nil {
				pOrder = t.BlockOrder()
			} 
			nOrder = t.BlockOrder()
			txs = append(txs, resource.NewTransaction(&t))
		}
		closeFunc()
		return txs
	}

	txs := readFunc()
	list := p.ResourceListWithOrder(txs, pOrder,nOrder)
	httputils.MustWriteJSON(w, 200, list)
}

func (api NetworkHandlerAPI) GetTransactionStatusByHashHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	key := vars["id"]

	status := "notfound"
	if found, _ := block.ExistsTransactionPool(api.storage, key); found {
		status = "submitted"
	}
	if found, _ := block.ExistsBlockTransaction(api.storage, key); found {
		status = "confirmed"
	}

	payload := resource.NewTransactionStatus(key, status)

	if httputils.IsEventStream(r) && status != "confirmed" {

		txStatusRenderFunc := func(args ...interface{}) ([]byte, error) {
			if len(args) <= 1 {
				return nil, fmt.Errorf("render: value is empty")
			}
			i := args[1]

			if i == nil {
				return nil, nil
			}

			switch v := i.(type) {
			case *block.TransactionPool:
				r := resource.NewTransactionStatus(key, "submitted")
				return json.Marshal(r.Resource())
			case *block.BlockTransaction:
				r := resource.NewTransactionStatus(key, "confirmed")
				return json.Marshal(r.Resource())
			case httputils.HALResource:
				return json.Marshal(v.Resource())
			}

			return json.Marshal(i)
		}

		es := NewEventStream(w, r, txStatusRenderFunc, DefaultContentType)
		es.Render(payload)
		es.Run(o.ResourceObserver,
			o.Event(o.NewCondition(o.Tx, o.TxHash, key)),
			o.Event(o.NewCondition(o.TxPool, o.TxHash, key)),
		)
		return
	}
	if payload.Status == "notfound" {
		httputils.MustWriteJSON(w, 404, payload)
	} else {
		httputils.MustWriteJSON(w, 200, payload)
	}
}
