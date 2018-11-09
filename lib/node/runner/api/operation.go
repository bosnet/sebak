package api

import (
	"fmt"
	"net/http"
	"strings"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/client"
	"boscoin.io/sebak/lib/common/observer"
	"boscoin.io/sebak/lib/errors"
	"boscoin.io/sebak/lib/network/httputils"
	"boscoin.io/sebak/lib/node/runner/api/resource"
	"boscoin.io/sebak/lib/transaction/operation"
	"github.com/gorilla/mux"
)

func (api NetworkHandlerAPI) GetOperationsByAccountHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	address := vars["id"]
	options, err := client.NewDefaultListOptionsFromQuery(r.URL.Query())
	if err != nil {
		http.Error(w, errors.InvalidQueryString.Error(), http.StatusBadRequest)
		return
	}

	oTypeStr := r.URL.Query().Get("type")
	if len(oTypeStr) > 0 && !operation.IsValidOperationType(oTypeStr) {
		http.Error(w, errors.InvalidQueryString.Error(), http.StatusBadRequest)
		return
	}

	oType := operation.OperationType(oTypeStr)
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
			if len(oType) == 0 || (len(oType) > 0 && t.Type == oType) {
				txs = append(txs, resource.NewOperation(&t))
			}
		}
		closeFunc()
		return txs
	}

	if httputils.IsEventStream(r) {
		event := fmt.Sprintf("source-%s", address)
		if len(oType) > 0 {
			event = fmt.Sprintf("source-type-%s%s", address, oType)
		}
		es := NewEventStream(w, r, renderEventStream, DefaultContentType)
		txs := readFunc()
		for _, tx := range txs {
			es.Render(tx)
		}
		es.Run(observer.BlockOperationObserver, event)
		return
	}

	if found, err := block.ExistsBlockAccount(api.storage, address); err != nil {
		httputils.WriteJSONError(w, err)
		return
	} else if !found {
		httputils.WriteJSONError(w, errors.BlockAccountDoesNotExists)
		return
	}

	txs := readFunc()
	self := r.URL.String()
	next := strings.Replace(resource.URLAccountOperations, "{id}", address, -1) + "?" + options.SetCursor(cursor).SetReverse(false).Encode()
	prev := strings.Replace(resource.URLAccountOperations, "{id}", address, -1) + "?" + options.SetReverse(true).Encode()
	list := resource.NewResourceList(txs, self, next, prev)

	httputils.MustWriteJSON(w, 200, list)
}
