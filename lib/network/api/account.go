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
)

func (api NetworkHandlerAPI) GetAccountHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	address := vars["id"]

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

	acc := resource.NewAccount(blk)

	if httputils.IsEventStream(r) {
		event := fmt.Sprintf("address-%s", address)
		es := NewEventStream(w, r, renderEventStream, DefaultContentType)
		es.Render(acc)
		es.Run(observer.BlockAccountObserver, event)
		return
	}

	if err := httputils.WriteJSON(w, 200, acc); err != nil {
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
	}
}
