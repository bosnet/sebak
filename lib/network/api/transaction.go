package api

import (
	"net/http"

	"boscoin.io/sebak/lib/common/observer"
	"boscoin.io/sebak/lib/network/httputils"

	"boscoin.io/sebak/lib/block"
	"github.com/gorilla/mux"
)

func (api NetworkHandlerAPI) GetTransactionsHandler(w http.ResponseWriter, r *http.Request) {

	readFunc := func(cnt int) []*block.BlockTransaction {
		var txs []*block.BlockTransaction
		iterFunc, closeFunc := block.GetBlockTransactions(api.storage, false)
		for {
			t, hasNext := iterFunc()
			if !hasNext || cnt == 0 {
				break
			}
			txs = append(txs, &t)
			cnt--
		}
		closeFunc()
		return txs
	}

	if httputils.IsEventStream(r) {
		event := "bt-saved"
		es := NewDefaultEventStream(w, r)
		txs := readFunc(maxNumberOfExistingData)
		for _, tx := range txs {
			es.Render(tx)
		}
		es.Run(observer.BlockTransactionObserver, event)
		return
	}

	txs := readFunc(-1) // -1 is infinte. TODO: Paging support makes better this space.

	if err := httputils.WriteJSON(w, 200, txs); err != nil {
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
		return
	}
}

func (api NetworkHandlerAPI) GetTransactionByHashHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	key := vars["txid"] //TODO: validate input

	found, err := block.ExistBlockTransaction(api.storage, key)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
		return
	}

	var payload interface{}

	if found {
		bt, err := block.GetBlockTransaction(api.storage, key)
		if err != nil {
			http.Error(w, "Error reading request body", http.StatusInternalServerError)
			return
		}
		payload = bt

	} else {
		bth, err := block.GetBlockTransactionHistory(api.storage, key)
		if err != nil {
			http.Error(w, "Error reading request body", http.StatusInternalServerError)
			return
		}
		payload = bth
	}

	if httputils.IsEventStream(r) {
		event := "bt-saved"
		es := NewDefaultEventStream(w, r)
		es.Render(payload)
		es.Run(observer.BlockTransactionObserver, event)
		return
	}

	if err := httputils.WriteJSON(w, 200, payload); err != nil {
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
	}
}
