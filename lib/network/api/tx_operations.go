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

func (api NetworkHandlerAPI) GetOperationsByTxHashHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	hash := vars["id"]

	if hash == "" {
		http.NotFound(w, r) //TODO(anarcher): 404-JSON
		return
	}

	iteratorOptions, err := storage.NewDefaultListOptionsFromQuery(r.URL.Query())
	if err != nil {
		http.Error(w, errors.ErrorInvalidQueryString.Error(), http.StatusBadRequest)
		return
	}

	if httputils.IsEventStream(r) {
		event := fmt.Sprintf("txhash-%s", hash)
		es := NewEventStream(w, r, renderEventStream, DefaultContentType)
		ops, _ := api.getOperationsByTxHash(hash, iteratorOptions)
		for _, op := range ops {
			es.Render(op)
		}
		es.Run(observer.BlockOperationObserver, event)
		return
	}

	ops, cursor := api.getOperationsByTxHash(hash, iteratorOptions)

	self := r.URL.String()
	next := GetTransactionOperationsHandlerPattern + "?" + "reverse=false&cursor=" + string(cursor)
	prev := GetTransactionOperationsHandlerPattern + "?" + "reverse=true&cursor=" + string(cursor)
	list := resource.NewResourceList(ops, self, next, prev)

	if err := httputils.WriteJSON(w, 200, list); err != nil {
		httputils.WriteJSONError(w, err)
		return
	}
}

func (api NetworkHandlerAPI) getOperationsByTxHash(txHash string, options storage.ListOptions) (txs []resource.Resource, cursor []byte) {
	iterFunc, closeFunc := block.GetBlockOperationsByTxHash(api.storage, txHash, options)
	for {
		o, hasNext, c := iterFunc()
		cursor = c
		if !hasNext {
			break
		}
		txs = append(txs, resource.NewOperation(&o))
	}
	closeFunc()
	return
}
