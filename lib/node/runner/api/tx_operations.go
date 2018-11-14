package api

import (
	"fmt"
	"net/http"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/common/observer"
	"boscoin.io/sebak/lib/errors"
	"boscoin.io/sebak/lib/network/httputils"
	"boscoin.io/sebak/lib/node/runner/api/resource"
	"boscoin.io/sebak/lib/storage"
	"github.com/gorilla/mux"
)

func (api NetworkHandlerAPI) GetOperationsByTxHashHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	hash := vars["id"]

	if hash == "" {
		httputils.WriteJSONError(w, errors.BlockTransactionDoesNotExists)
		return
	}

	p, err := NewPageQuery(r)
	if err != nil {
		httputils.WriteJSONError(w, err)
		return
	}

	options := p.ListOptions()

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
		httputils.WriteJSONError(w, errors.BlockTransactionDoesNotExists)
		return
	}

	list := p.ResourceList(ops, cursor)
	httputils.MustWriteJSON(w, 200, list)
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
