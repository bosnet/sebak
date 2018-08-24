package sebak

import (
	"fmt"
	"net/http"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/error"
	"boscoin.io/sebak/lib/observer"
	"github.com/gorilla/mux"
)

const GetAccountHandlerPattern = "/account/{address}"

func (api NetworkHandlerAPI) GetAccountHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var blk *block.BlockAccount
		var err error
		vars := mux.Vars(r)
		address := vars["address"]
		if blk, err = block.GetBlockAccount(api.storage, address); err != nil {
			http.Error(w, "Error reading request body", http.StatusInternalServerError)
			return
		}

		switch r.Header.Get("Accept") {
		case "text/event-stream":
			var readyChan = make(chan struct{})
			// Trigger event for data already stored in the storage
			iterateId := sebakcommon.GetUniqueIDFromUUID()
			go func() {
				<-readyChan
				observer.BlockAccountObserver.Trigger(fmt.Sprintf("iterate-%s", iterateId), blk)
			}()

			callBackFunc := func(args ...interface{}) (account []byte, err error) {
				blk := args[1].(*block.BlockAccount)
				if account, err = blk.Serialize(); err != nil {
					return []byte{}, sebakerror.ErrorBlockAccountDoesNotExists
				}
				return account, nil
			}

			event := fmt.Sprintf("iterate-%s", iterateId)
			event += " " + fmt.Sprintf("address-%s", address)
			streaming(observer.BlockAccountObserver, r, w, event, callBackFunc, readyChan)
		default:
			var s []byte
			if s, err = blk.Serialize(); err != nil {
				http.Error(w, "Error reading request body", http.StatusInternalServerError)
				return
			}
			if _, err = w.Write(s); err != nil {
				http.Error(w, "Error reading request body", http.StatusInternalServerError)
				return
			}
		}
	}
}

const GetAccountTransactionsHandlerPattern = "/account/{address}/transactions"

func (api NetworkHandlerAPI) GetAccountTransactionsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		vars := mux.Vars(r)
		address := vars["address"]

		var err error
		var s []byte

		switch r.Header.Get("Accept") {
		case "text/event-stream":
			var readyChan = make(chan struct{})
			iterateId := sebakcommon.GetUniqueIDFromUUID()
			go func() {
				<-readyChan
				count := maxNumberOfExistingData
				iterFunc, closeFunc := GetBlockTransactionsByAccount(api.storage, address, false)
				for {
					bt, hasNext := iterFunc()
					count--
					if !hasNext || count < 0 {
						break
					}
					observer.BlockTransactionObserver.Trigger(fmt.Sprintf("iterate-%s", iterateId), &bt)
				}
				closeFunc()
			}()

			callBackFunc := func(args ...interface{}) (btSerialized []byte, err error) {
				bt := args[1].(*BlockTransaction)
				if btSerialized, err = bt.Serialize(); err != nil {
					return []byte{}, sebakerror.ErrorBlockTransactionDoesNotExists
				}
				return btSerialized, nil
			}

			event := fmt.Sprintf("iterate-%s", iterateId)
			event += " " + fmt.Sprintf("source-%s", address)
			streaming(observer.BlockTransactionObserver, r, w, event, callBackFunc, readyChan)
		default:

			var btl []BlockTransaction
			iterFunc, closeFunc := GetBlockTransactionsByAccount(api.storage, address, false)
			for {
				bt, hasNext := iterFunc()
				if !hasNext {
					break
				}
				btl = append(btl, bt)
			}
			closeFunc()

			s, err = sebakcommon.EncodeJSONValue(btl)

			if _, err = w.Write(s); err != nil {
				http.Error(w, "Error reading request body", http.StatusInternalServerError)
				return
			}
		}
	}
}

const GetAccountOperationsHandlerPattern = "/account/{address}/operations"

func (api NetworkHandlerAPI) GetAccountOperationsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		address := vars["address"]

		var err error
		var s []byte

		switch r.Header.Get("Accept") {
		case "text/event-stream":
			var readyChan = make(chan struct{})
			iterateId := sebakcommon.GetUniqueIDFromUUID()
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
					return []byte{}, sebakerror.ErrorBlockTransactionDoesNotExists
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

			s, err = sebakcommon.EncodeJSONValue(bol)

			if _, err = w.Write(s); err != nil {
				http.Error(w, "Error reading request body", http.StatusInternalServerError)
				return
			}
		}
	}
}
