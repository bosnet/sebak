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

func (api NetworkHandlerAPI) GetAccountHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	address := vars["id"]

	readFunc := func() (payload interface{}, err error) {
		found, err := block.ExistsBlockAccount(api.storage, address)
		if err != nil {
			return nil, err
		}
		if !found {
			return nil, errors.ErrorBlockAccountDoesNotExists
		}
		ba, err := block.GetBlockAccount(api.storage, address)
		if err != nil {
			return nil, err
		}
		payload = resource.NewAccount(ba)
		return payload, nil
	}

	if httputils.IsEventStream(r) {
		event := fmt.Sprintf("address-%s", address)
		es := NewEventStream(w, r, renderEventStream, DefaultContentType)
		payload, err := readFunc()
		if err == nil {
			es.Render(payload)
		}
		es.Run(observer.BlockAccountObserver, event)
		return
	}

	payload, err := readFunc()
	if err != nil {
		httputils.WriteJSONError(w, err)
		return
	}

	httputils.MustWriteJSON(w, 200, payload)
}
