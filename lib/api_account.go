package sebak

import (
	"fmt"
	"net/http"

	"boscoin.io/sebak/lib/error"
	"boscoin.io/sebak/lib/observer"
	"boscoin.io/sebak/lib/storage"
	"github.com/gorilla/mux"
)

func GetAccountHandler(s *sebakstorage.LevelDBBackend) http.HandlerFunc {
	storage := s

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
		if ba, err = GetBlockAccount(storage, address); err != nil {
			http.Error(w, "Error reading request body", http.StatusInternalServerError)
			return
		}

		var s []byte
		if s, err = ba.Serialize(); err != nil {
			http.Error(w, "Error reading request body", http.StatusInternalServerError)
			return
		}

		switch r.Header.Get("Accept") {
		case "text/event-stream":
			event := fmt.Sprintf("saved-%s", address)
			callBackFunc := func(args ...interface{}) (account []byte, err error) {
				ba := args[0].(*BlockAccount)
				if account, err = ba.Serialize(); err != nil {
					return []byte{}, sebakerror.ErrorBlockAccountDoesNotExists
				}
				return account, nil
			}
			streaming(observer.BlockAccountObserver, w, event, callBackFunc, s)
		default:
			if _, err = w.Write(s); err != nil {
				http.Error(w, "Error reading request body", http.StatusInternalServerError)
				return
			}
		}
	}
}
