package api

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/common/observer"
	"boscoin.io/sebak/lib/error"
	"boscoin.io/sebak/lib/network/httputils"
	"boscoin.io/sebak/lib/storage"
)

func (api NetworkHandlerAPI) GetAccountHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	address := vars["address"]

	var (
		blk *block.BlockAccount
		err error
	)

	if blk, err = block.GetBlockAccount(api.storage, address); err != nil {
		if err == errors.ErrorStorageRecordDoesNotExist {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		} else {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}

	if httputils.IsEventStream(r) {
		event := fmt.Sprintf("address-%s", address)
		es := NewDefaultEventStream(w, r)
		es.Render(blk)
		es.Run(observer.BlockAccountObserver, event)
		return
	}

	if err := httputils.WriteJSON(w, 200, blk); err != nil {
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
	}
}

func (api NetworkHandlerAPI) GetAccountTransactionsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	address := vars["address"]

	readFunc := func(cnt int) []*block.BlockTransaction {
		var txs []*block.BlockTransaction
		iterFunc, closeFunc := block.GetBlockTransactionsByAccount(api.storage, address, &storage.IteratorOptions{Reverse: false})
		for {
			t, hasNext, _ := iterFunc()
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
		event := fmt.Sprintf("bt-source-%s", address)
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

func (api NetworkHandlerAPI) GetAccountOperationsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	address := vars["address"]

	readFunc := func(cnt int) []*block.BlockOperation {
		var txs []*block.BlockOperation
		iterFunc, closeFunc := block.GetBlockOperationsBySource(api.storage, address, &storage.IteratorOptions{Reverse: false})
		for {
			t, hasNext, _ := iterFunc()
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
		event := fmt.Sprintf("bo-source-%s", address)
		es := NewDefaultEventStream(w, r)
		txs := readFunc(maxNumberOfExistingData)
		for _, tx := range txs {
			es.Render(tx)
		}
		es.Run(observer.BlockOperationObserver, event)
		return
	}

	txs := readFunc(-1) //TODO paging support
	if err := httputils.WriteJSON(w, 200, txs); err != nil {
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
		return
	}
}
