package api

import (
	"net/http"

	"boscoin.io/sebak/lib/common/observer"
	"boscoin.io/sebak/lib/network/api/resource"
	"boscoin.io/sebak/lib/network/httputils"

	"boscoin.io/sebak/lib/block"
	"fmt"
	"github.com/gorilla/mux"
)

func (api NetworkHandlerAPI) GetTransactionsHandler(w http.ResponseWriter, r *http.Request) {

	iteratorOptions := parseQueryString(r)
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
		event := "bt-saved"
		es := NewEventStream(w, r, renderEventStream, DefaultContentType)
		txs := readFunc()
		for _, tx := range txs {
			es.Render(tx)
		}
		es.Run(observer.BlockTransactionObserver, event)
		return
	}

	txs := readFunc() // -1 is infinte. TODO: Paging support makes better this space.

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
	key := vars["id"] //TODO: validate input

	readFunc := func() (payload interface{}, err error) {
		found, err := block.ExistBlockTransaction(api.storage, key)
		if err != nil {
			//http.Error(w, "Error reading request body", http.StatusInternalServerError)
			return
		}

		if found {
			bt, err := block.GetBlockTransaction(api.storage, key)
			if err != nil {
				//http.Error(w, "Error reading request body", http.StatusInternalServerError)
				return nil, err
			}
			payload = bt

		} else {
			bth, err := block.GetBlockTransactionHistory(api.storage, key)
			if err != nil {
				//http.Error(w, "Error reading request body", http.StatusInternalServerError)
				return nil, err
			}
			payload = bth
		}
		return
	}

	if httputils.IsEventStream(r) {
		event := "bt-saved"
		es := NewDefaultEventStream(w, r)
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
		if err := httputils.WriteJSON(w, 200, payload); err != nil {
			http.Error(w, "Error reading request body", http.StatusInternalServerError)
		}
	}

}

func (api NetworkHandlerAPI) GetTransactionsByAccountHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	address := vars["id"]
	iteratorOptions := parseQueryString(r)
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
		event := fmt.Sprintf("bt-source-%s", address)
		es := NewEventStream(w, r, renderEventStream, DefaultContentType)
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
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
		return
	}
}
