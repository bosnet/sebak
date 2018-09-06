package sebak

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/error"
	"boscoin.io/sebak/lib/httputils"
	"boscoin.io/sebak/lib/observer"
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

	readFunc := func(cnt int) []BlockTransaction {
		var txs []BlockTransaction
		iterFunc, closeFunc := GetBlockTransactionsByAccount(api.storage, address, false)
		for {
			t, hasNext := iterFunc()
			cnt--
			if !hasNext || cnt == 0 {
				break
			}
			txs = append(txs, t)
		}
		closeFunc()
		return txs
	}

	if httputils.IsEventStream(r) {
		event := fmt.Sprintf("source-%s", address)
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

	var err error
	var s []byte

	switch r.Header.Get("Accept") {
	case "text/event-stream":
		var readyChan = make(chan struct{})
		iterateId := common.GetUniqueIDFromUUID()
		go func() {
			<-readyChan
			count := maxNumberOfExistingData
			iterFunc, closeFunc := GetBlockOperationsBySource(api.storage, address, false)
			for {
				bo, hasNext := iterFunc()
				count--
				if !hasNext || count < 0 {
					break
				}
				observer.BlockOperationObserver.Trigger(fmt.Sprintf("iterate-%s", iterateId), &bo)
			}
			closeFunc()
		}()

		callBackFunc := func(args ...interface{}) (boSerialized []byte, err error) {
			bo := args[1].(*BlockOperation)
			if boSerialized, err = bo.Serialize(); err != nil {
				return []byte{}, errors.ErrorBlockTransactionDoesNotExists
			}
			return boSerialized, nil
		}
		event := fmt.Sprintf("iterate-%s", iterateId)
		event += " " + fmt.Sprintf("source-%s", address)
		streaming(observer.BlockOperationObserver, r, w, event, callBackFunc, readyChan)
	default:
		var bol []BlockOperation
		iterFunc, closeFunc := GetBlockOperationsBySource(api.storage, address, false)
		for {
			bo, hasNext := iterFunc()
			if !hasNext {
				break
			}
			bol = append(bol, bo)
		}
		closeFunc()

		s, err = common.EncodeJSONValue(bol)

		if _, err = w.Write(s); err != nil {
			http.Error(w, "Error reading request body", http.StatusInternalServerError)
			return
		}
	}
}
