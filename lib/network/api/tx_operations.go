package api

import (
	"fmt"
	"net/http"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/common/observer"
	"boscoin.io/sebak/lib/network/api/resource"
	"boscoin.io/sebak/lib/network/httputils"
	"boscoin.io/sebak/lib/storage"
	"github.com/gorilla/mux"
)

func (api NetworkHandlerAPI) GetOperationsByTxHashHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	hash := vars["id"]

	if hash == "" {
		http.NotFound(w, r) //TODO(anarcher): 404-JSON
		return
	}

	iterOps := parseQueryString(r)

	if httputils.IsEventStream(r) {
		event := fmt.Sprintf("txhash-%s", hash)
		es := NewEventStream(w, r, renderEventStream, DefaultContentType)
		ops, _ := api.getOperationsByTxHash(hash, iterOps)
		for _, op := range ops {
			es.Render(op)
		}
		es.Run(observer.BlockOperationObserver, event)
		return
	}

	ops, cursor := api.getOperationsByTxHash(hash, iterOps)

	self := r.URL.String()
	next := GetTransactionOperationsHandlerPattern + "?" + "reverse=false&cursor=" + string(cursor)
	prev := GetTransactionOperationsHandlerPattern + "?" + "reverse=true&cursor=" + string(cursor)
	list := resource.NewResourceList(ops, self, next, prev)

	if err := httputils.WriteJSON(w, 200, list); err != nil {
		httputils.WriteJSONError(w, err)
		return
	}
}

func (api NetworkHandlerAPI) getOperationsByTxHash(txHash string, iterOps *storage.IteratorOptions) (txs []resource.Resource, cursor []byte) {
	iterFunc, closeFunc := block.GetBlockOperationsByTxHash(api.storage, txHash, iterOps)
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
