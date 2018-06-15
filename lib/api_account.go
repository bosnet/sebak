package sebak

import (
	"fmt"
	"net/http"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/error"
	"boscoin.io/sebak/lib/observer"
	"boscoin.io/sebak/lib/storage"
	"github.com/gorilla/mux"
)

const GetAccountHandlerPattern = "/account/{address}"

func GetAccountHandler(storage *sebakstorage.LevelDBBackend) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {

		vars := mux.Vars(r)
		address := vars["address"]
		if found, err := ExistBlockAccount(storage, address); err != nil {
			http.Error(w, "Error reading request body", http.StatusInternalServerError)
			return
		} else if !found {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}

		var err error
		var ba *BlockAccount

		switch r.Header.Get("Accept") {
		case "text/event-stream":

			var readyChan = make(chan struct{})

			iterateId := sebakcommon.GetUniqueIDFromUUID()
			go func() {
				<-readyChan
				if ba, err = GetBlockAccount(storage, address); err != nil {
					http.Error(w, "Error reading request body", http.StatusInternalServerError)
					return
				}
				observer.BlockAccountObserver.Trigger(fmt.Sprintf("iterate-%s", iterateId), ba)
			}()

			event := fmt.Sprintf("iterate-%s", iterateId)
			event += " " + fmt.Sprintf("address-%s", address)
			callBackFunc := func(args ...interface{}) (account []byte, err error) {
				//typedEvent := args[0].(string)
				ba := args[1].(*BlockAccount)
				if account, err = ba.Serialize(); err != nil {
					return []byte{}, sebakerror.ErrorBlockAccountDoesNotExists
				}
				return account, nil
			}
			streaming(observer.BlockAccountObserver, w, event, callBackFunc, readyChan)
		default:
			if ba, err = GetBlockAccount(storage, address); err != nil {
				http.Error(w, "Error reading request body", http.StatusInternalServerError)
				return
			}

			var s []byte
			if s, err = ba.Serialize(); err != nil {
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

func GetAccountTransactionsHandler(storage *sebakstorage.LevelDBBackend) http.HandlerFunc {

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
				iterFunc, closeFunc := GetBlockTransactionsByAccount(storage, address, false)
				for {
					bt, hasNext := iterFunc()
					if !hasNext {
						break
					}
					observer.BlockTransactionObserver.Trigger(fmt.Sprintf("iterate-%s", iterateId), &bt)
				}
				closeFunc()
			}()

			event := fmt.Sprintf("iterate-%s", iterateId)
			event += " " + fmt.Sprintf("source-%s", address)
			callBackFunc := func(args ...interface{}) (btSerialized []byte, err error) {
				//typedEvent := args[0].(string)
				bt := args[1].(*BlockTransaction)
				if btSerialized, err = bt.Serialize(); err != nil {
					return []byte{}, sebakerror.ErrorBlockTransactionDoesNotExists
				}
				return btSerialized, nil
			}
			streaming(observer.BlockTransactionObserver, w, event, callBackFunc, readyChan)
		default:

			var btl []BlockTransaction
			iterFunc, closeFunc := GetBlockTransactionsByAccount(storage, address, false)
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

func GetAccountOperationsHandler(storage *sebakstorage.LevelDBBackend) http.HandlerFunc {

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
				iterFunc, closeFunc := GetBlockOperationsBySource(storage, address, false)
				for {
					bo, hasNext := iterFunc()
					if !hasNext {
						break
					}
					observer.BlockOperationObserver.Trigger(fmt.Sprintf("iterate-%s", iterateId), &bo)
				}
				closeFunc()
			}()

			event := fmt.Sprintf("iterate-%s", iterateId)
			event += " " + fmt.Sprintf("source-%s", address)
			callBackFunc := func(args ...interface{}) (boSerialized []byte, err error) {

				//typedEvent := args[0].(string)
				bo := args[1].(*BlockOperation)
				if boSerialized, err = bo.Serialize(); err != nil {
					return []byte{}, sebakerror.ErrorBlockTransactionDoesNotExists
				}
				return boSerialized, nil
			}
			streaming(observer.BlockOperationObserver, w, event, callBackFunc, readyChan)
		default:

			var bol []BlockOperation
			iterFunc, closeFunc := GetBlockOperationsBySource(storage, address, false)
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
