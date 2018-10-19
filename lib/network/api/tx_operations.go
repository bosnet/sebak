package api

import (
	"fmt"
	"net/http"
	"strings"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/common/observer"
	"boscoin.io/sebak/lib/error"
	"boscoin.io/sebak/lib/network/api/resource"
	"boscoin.io/sebak/lib/network/httputils"
	"boscoin.io/sebak/lib/storage"
	"github.com/gorilla/mux"
)

func (api NetworkHandlerAPI) GetOperationsByTxHashHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	hash := vars["id"]

	if hash == "" {
		httputils.WriteJSONError(w, errors.ErrorBlockTransactionDoesNotExists)
		return
	}

	options, err := storage.NewDefaultListOptionsFromQuery(r.URL.Query())
	if err != nil {
		http.Error(w, errors.ErrorInvalidQueryString.Error(), http.StatusBadRequest)
		return
	}

	if httputils.IsEventStream(r) {
		event := fmt.Sprintf("txhash-%s", hash)
		es := NewEventStream(w, r, renderEventStream, DefaultContentType)
		ops, _ := api.getOperationsByTxHash(hash, options)
		for _, op := range ops {
			es.Render(op)
		}
		es.Run(observer.BlockOperationObserver, event)
		return
	}

	ops, cursor := api.getOperationsByTxHash(hash, options)
	if len(ops) < 1 {
		httputils.WriteJSONError(w, errors.ErrorBlockTransactionDoesNotExists)
		return
	}

	self := r.URL.String()
	next := strings.Replace(resource.URLTransactionOperations, "{id}", hash, -1) + "?" + options.SetCursor(cursor).SetReverse(false).Encode()
	prev := strings.Replace(resource.URLTransactionOperations, "{id}", hash, -1) + "?" + options.SetReverse(true).Encode()
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
