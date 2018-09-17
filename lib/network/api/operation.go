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
	"boscoin.io/sebak/lib/storage"
)

func (api NetworkHandlerAPI) GetOperationsByAccountHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	address := vars["id"]
	options, err := storage.NewDefaultListOptionsFromQuery(r.URL.Query())
	if err != nil {
		http.Error(w, errors.ErrorInvalidQueryString.Error(), http.StatusBadRequest)
		return
	}
	var cursor []byte
	readFunc := func() []resource.Resource {
		var txs []resource.Resource
		iterFunc, closeFunc := block.GetBlockOperationsBySource(api.storage, address, options)
		for {
			t, hasNext, c := iterFunc()
			cursor = c
			if !hasNext {
				break
			}
			txs = append(txs, resource.NewOperation(&t))
		}
		closeFunc()
		return txs
	}

	if httputils.IsEventStream(r) {
		event := fmt.Sprintf("source-%s", address)
		es := NewEventStream(w, r, renderEventStream, DefaultContentType)
		txs := readFunc()
		for _, tx := range txs {
			es.Render(tx)
		}
		es.Run(observer.BlockOperationObserver, event)
		return
	}

	txs := readFunc() //TODO paging support
	self := r.URL.String()
	next := GetAccountOperationsHandlerPattern + "?" + options.SetCursor(cursor).SetReverse(false).URLValues().Encode()
	prev := GetAccountOperationsHandlerPattern + "?" + options.SetReverse(true).URLValues().Encode()
	list := resource.NewResourceList(txs, self, next, prev)

	if err := httputils.WriteJSON(w, 200, list); err != nil {
		httputils.WriteJSONError(w, err)
		return
	}
}
