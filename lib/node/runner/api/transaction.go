package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/common/observer"
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
	var cursor []byte
	readFunc := func() []resource.Resource {
		var txs []resource.Resource
		iterFunc, closeFunc := block.GetBlockTransactions(api.storage, options)
		for {
			t, hasNext, c := iterFunc()
			cursor = c
			if !hasNext {
				break
			}
			txs = append(txs, resource.NewTransaction(&t))
		}
		closeFunc()
		return txs
	}

	if httputils.IsEventStream(r) {
		event := "saved"
		es := NewEventStream(w, r, renderEventStream, DefaultContentType)
		options.SetLimit(10)
		txs := readFunc()
		if len(txs) > 0 {
			for _, tx := range txs {
				es.Render(tx)
			}
		} else {
			es.Render(nil)
		}
		es.Run(observer.BlockTransactionObserver, event)
		return
	}

	txs := readFunc()

	list := p.ResourceList(txs, cursor)
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

	if httputils.IsEventStream(r) {
		event := fmt.Sprintf("hash-%s", key)
		es := NewEventStream(w, r, renderEventStream, DefaultContentType)
		payload, err := readFunc()
		if err == nil {
			es.Render(payload)
		} else {
			es.Render(nil)
		}
		es.Run(observer.BlockTransactionObserver, event)
		return
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

	var options = p.ListOptions()
	var cursor []byte
	readFunc := func() []resource.Resource {
		var txs []resource.Resource
		iterFunc, closeFunc := block.GetBlockTransactionsByAccount(api.storage, address, options)
		for {
			t, hasNext, c := iterFunc()
			cursor = c
			if !hasNext {
				break
			}
			txs = append(txs, resource.NewTransaction(&t))
		}
		closeFunc()
		return txs
	}

	if httputils.IsEventStream(r) {
		event := fmt.Sprintf("source-%s", address)
		es := NewEventStream(w, r, renderEventStream, DefaultContentType)
		options.SetLimit(10)
		txs := readFunc()
		if len(txs) > 0 {
			for _, tx := range txs {
				es.Render(tx)
			}
		} else {
			es.Render(nil)
		}
		es.Run(observer.BlockTransactionObserver, event)
		return
	}

	txs := readFunc()
	list := p.ResourceList(txs, cursor)
	httputils.MustWriteJSON(w, 200, list)
}

func (api NetworkHandlerAPI) GetTransactionStatusByHashHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	key := vars["id"]

	status := "notfound"
	if found, _ := block.ExistsTransactionPool(api.storage, key); found {
		status = "submitted"
	} else if found, _ = block.ExistsBlockTransaction(api.storage, key); found {
		status = "confirmed"
	}

	payload := resource.NewTransactionStatus(key, status)

	if httputils.IsEventStream(r) && status != "confirmed" {
		event := fmt.Sprintf("hash-%s", key)
		event += " " + fmt.Sprintf("pushed-%s", key)

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
		es.Run(observer.BlockTransactionObserver, event)
		return
	}
	if payload.Status == "notfound" {
		httputils.MustWriteJSON(w, 404, payload)
	} else {
		httputils.MustWriteJSON(w, 200, payload)
	}
}
