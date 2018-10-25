package api

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/common/observer"
	"boscoin.io/sebak/lib/errors"
	"boscoin.io/sebak/lib/network/httputils"
	"boscoin.io/sebak/lib/node/runner/api/resource"
)

func (api NetworkHandlerAPI) GetTransactionHistoryHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	key := vars["id"]

	readFunc := func() (payload interface{}, err error) {
		found, err := block.ExistsBlockTransactionHistory(api.storage, key)
		if err != nil {
			return nil, err
		}
		if !found {
			return nil, errors.ErrorBlockTransactionDoesNotExists
		}
		bt, err := block.GetBlockTransactionHistory(api.storage, key)
		if err != nil {
			return nil, err
		}
		payload = resource.NewTransactionHistory(&bt)
		return payload, nil
	}

	if httputils.IsEventStream(r) {
		event := fmt.Sprintf("hash-%s", key)
		es := NewEventStream(w, r, renderEventStream, DefaultContentType)
		payload, err := readFunc()
		if err == nil {
			es.Render(payload)
		}
		es.Run(observer.BlockTransactionHistoryObserver, event)
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
