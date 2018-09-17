package api

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/common/observer"
	"boscoin.io/sebak/lib/error"
	"boscoin.io/sebak/lib/network/api/resource"
	"boscoin.io/sebak/lib/network/httputils"
	"boscoin.io/sebak/lib/storage"
)

func (api NetworkHandlerAPI) GetTransactionsHandler(w http.ResponseWriter, r *http.Request) {
	iteratorOptions, err := storage.NewDefaultListOptionsFromQuery(r.URL.Query())
	if err != nil {
		http.Error(w, errors.ErrorInvalidQueryString.Error(), http.StatusBadRequest)
		return
	}
	var cursor []byte
	readFunc := func() []resource.Resource {
		var txs []resource.Resource
		iterFunc, closeFunc := block.GetBlockTransactions(api.storage, iteratorOptions)
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
		iteratorOptions.SetLimit(10)
		txs := readFunc()
		for _, tx := range txs {
			es.Render(tx)
		}
		es.Run(observer.BlockTransactionObserver, event)
		return
	}

	txs := readFunc()

	self := r.URL.String()
	next := GetTransactionsHandlerPattern + "?" + "reverse=false&cursor=" + string(cursor)
	prev := GetTransactionsHandlerPattern + "?" + "reverse=true&cursor=" + string(cursor)
	list := resource.NewResourceList(txs, self, next, prev)

	if err := httputils.WriteJSON(w, 200, list); err != nil {
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
		return
	}
}

func (api NetworkHandlerAPI) GetTransactionByHashHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	key := vars["id"]

	readFunc := func() (payload interface{}, err error) {
		found, err := block.ExistBlockTransaction(api.storage, key)
		if err != nil {
			return nil, err
		}
		if !found {
			return nil, errors.ErrorBlockTransactionDoesNotExists
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
		}
		es.Run(observer.BlockTransactionObserver, event)
		return
	}
	payload, err := readFunc()
	if err == nil {
		if err := httputils.WriteJSON(w, 200, payload); err != nil {
			http.Error(w, "Error reading request body", http.StatusInternalServerError)
		}
	} else {
		if err := httputils.WriteJSON(w, httputils.StatusCode(err), err); err != nil {
			http.Error(w, "Error reading request body", http.StatusInternalServerError)
		}
	}
}

func (api NetworkHandlerAPI) GetTransactionsByAccountHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	address := vars["id"]
	iteratorOptions, err := storage.NewDefaultListOptionsFromQuery(r.URL.Query())
	if err != nil {
		http.Error(w, errors.ErrorInvalidQueryString.Error(), http.StatusBadRequest)
		return
	}
	var cursor []byte
	readFunc := func() []resource.Resource {
		var txs []resource.Resource
		iterFunc, closeFunc := block.GetBlockTransactionsByAccount(api.storage, address, iteratorOptions)
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
		iteratorOptions.SetLimit(10)
		txs := readFunc()
		for _, tx := range txs {
			es.Render(tx)
		}
		es.Run(observer.BlockTransactionObserver, event)
		return
	}

	txs := readFunc()
	self := r.URL.String()
	next := GetAccountTransactionsHandlerPattern + "?" + "reverse=false&cursor=" + string(cursor)
	prev := GetAccountTransactionsHandlerPattern + "?" + "reverse=true&cursor=" + string(cursor)
	list := resource.NewResourceList(txs, self, next, prev)

	if err := httputils.WriteJSON(w, 200, list); err != nil {
		httputils.WriteJSONError(w, err)
		return
	}
}
