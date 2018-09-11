package api

import (
	"net/http"

	"boscoin.io/sebak/lib/common/observer"
	"boscoin.io/sebak/lib/network/api/resource"
	"boscoin.io/sebak/lib/network/httputils"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/storage"
	"github.com/gorilla/mux"
)

func (api NetworkHandlerAPI) GetTransactionsHandler(w http.ResponseWriter, r *http.Request) {

	readFunc := func(cnt int) []resource.Resource {
		var txs []resource.Resource
		iterFunc, closeFunc := block.GetBlockTransactions(api.storage, &storage.IteratorOptions{Reverse: false})
		for {
			t, hasNext, _ := iterFunc()
			if !hasNext || cnt == 0 {
				break
			}
			txs = append(txs, resource.NewTransaction(&t))
			cnt--
		}
		closeFunc()
		return txs
	}

	if httputils.IsEventStream(r) {
		event := "bt-saved"
		es := NewEventStream(w, r, renderEventStream, DefaultContentType)
		txs := readFunc(maxNumberOfExistingData)
		for _, tx := range txs {
			es.Render(tx)
		}
		es.Run(observer.BlockTransactionObserver, event)
		return
	}

	txs := readFunc(-1) // -1 is infinte. TODO: Paging support makes better this space.

	list := resource.NewResourceList(txs, GetTransactionsHandlerPattern)

	if err := httputils.WriteJSON(w, 200, list); err != nil {
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

	var apiResource resource.Resource

	if found {
		bt, err := block.GetBlockTransaction(api.storage, key)
		if err != nil {
			http.Error(w, "Error reading request body", http.StatusInternalServerError)
			return
		}
		apiResource = resource.NewTransaction(&bt)

	} else {
		bth, err := block.GetBlockTransactionHistory(api.storage, key)
		if err != nil {
			http.Error(w, "Error reading request body", http.StatusInternalServerError)
			return
		}
		apiResource = resource.NewTransactionHistory(&bth)
	}

	if httputils.IsEventStream(r) {
		event := "bt-saved"
		es := NewEventStream(w, r, renderEventStream, DefaultContentType)
		es.Render(apiResource)
		es.Run(observer.BlockTransactionObserver, event)
		return
	}

	if err := httputils.WriteJSON(w, 200, apiResource); err != nil {
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
	}
}
