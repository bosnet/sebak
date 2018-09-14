package api

import (
	"fmt"
	"net/http"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/common/observer"
	"boscoin.io/sebak/lib/network/api/resource"
	"boscoin.io/sebak/lib/network/httputils"
	"github.com/gorilla/mux"
)

func (api NetworkHandlerAPI) GetOperationsByAccountHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	address := vars["id"]
	iteratorOptions := parseQueryString(r)
	var cursor []byte
	readFunc := func() []resource.Resource {
		var txs []resource.Resource
		iterFunc, closeFunc := block.GetBlockOperationsBySource(api.storage, address, iteratorOptions)
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
	next := GetAccountOperationsHandlerPattern + "?" + "reverse=false&cursor=" + string(cursor)
	prev := GetAccountOperationsHandlerPattern + "?" + "reverse=true&cursor=" + string(cursor)
	list := resource.NewResourceList(txs, self, next, prev)

	if err := httputils.WriteJSON(w, 200, list); err != nil {
		httputils.WriteJSONError(w, err)
		return
	}
}
