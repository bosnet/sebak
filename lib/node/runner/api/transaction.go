package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"

	"boscoin.io/sebak/lib/block"
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

	var options = p.ListOptions()
	var firstCursor []byte
	var cursor []byte
	var txs []resource.Resource
	iterFunc, closeFunc := block.GetBlockTransactions(api.storage, options)
	for {
		t, hasNext, c := iterFunc()
		if !hasNext {
			break
		}
		cursor = append([]byte{}, c...)
		if len(firstCursor) == 0 {
			firstCursor = append(firstCursor, c...)
		}
		tp, err := block.GetTransactionPool(api.storage, t.Hash)
		if err != nil {
			httputils.WriteJSONError(w, err)
			return
		}
		txs = append(txs, resource.NewTransaction(&t, tp.Transaction()))
	}
	closeFunc()

	list := p.ResourceList(txs, firstCursor, cursor)
	httputils.MustWriteJSON(w, 200, list)
}

func (api NetworkHandlerAPI) GetTransactionByHashHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	key := vars["id"]

	found, err := block.ExistsBlockTransaction(api.storage, key)
	if err != nil {
		httputils.WriteJSONError(w, err)
		return
	}
	if !found {
		httputils.WriteJSONError(w, errors.BlockTransactionDoesNotExists)
		return
	}
	bt, err := block.GetBlockTransaction(api.storage, key)
	if err != nil {
		httputils.WriteJSONError(w, err)
		return
	}
	tp, err := block.GetTransactionPool(api.storage, bt.Hash)
	if err != nil {
		httputils.WriteJSONError(w, err)
		return
	}
	tx := resource.NewTransaction(&bt, tp.Transaction())

	httputils.MustWriteJSON(w, 200, tx)
}

func (api NetworkHandlerAPI) GetTransactionsByAccountHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	address := vars["id"]

	p, err := NewPageQuery(r)
	if err != nil {
		httputils.WriteJSONError(w, err)
		return
	}

	var options = p.ListOptions()
	var firstCursor []byte
	var cursor []byte
	var txs []resource.Resource
	iterFunc, closeFunc := block.GetBlockTransactionsByAccount(api.storage, address, options)
	for {
		t, hasNext, c := iterFunc()
		if !hasNext {
			break
		}
		cursor = append([]byte{}, c...)
		if len(firstCursor) == 0 {
			firstCursor = append(firstCursor, c...)
		}
		tp, err := block.GetTransactionPool(api.storage, t.Hash)
		if err != nil {
			httputils.WriteJSONError(w, err)
			return
		}
		txs = append(txs, resource.NewTransaction(&t, tp.Transaction()))
	}
	closeFunc()
	list := p.ResourceList(txs, firstCursor, cursor)
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
			o.Event(o.NewCondition(o.Tx, o.Identifier, key)),
			o.Event(o.NewCondition(o.TxPool, o.Identifier, key)),
		)
		return
	}
	if payload.Status == "notfound" {
		httputils.MustWriteJSON(w, 404, payload)
	} else {
		httputils.MustWriteJSON(w, 200, payload)
	}
}
