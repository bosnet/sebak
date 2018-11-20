package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/common/observer"
	"boscoin.io/sebak/lib/errors"
	"boscoin.io/sebak/lib/network/httputils"
	"boscoin.io/sebak/lib/node/runner/api/resource"
	"boscoin.io/sebak/lib/transaction/operation"
)

func (api NetworkHandlerAPI) GetOperationsByAccountHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	address := vars["id"]

	p, err := NewPageQuery(r)
	if err != nil {
		httputils.WriteJSONError(w, err)
		return
	}

	options := p.ListOptions()

	oTypeStr := r.URL.Query().Get("type")
	if len(oTypeStr) > 0 && !operation.IsValidOperationType(oTypeStr) {
		httputils.WriteJSONError(w, errors.InvalidQueryString)
		return
	}

	blockCache := map[ /* block.Height */ uint64]*block.Block{}
	oType := operation.OperationType(oTypeStr)
	var cursor []byte
	readFunc := func() []resource.Resource {
		var txs []resource.Resource

		var iterFunc func() (block.BlockOperation, bool, []byte)
		var closeFunc func()
		if len(oType) > 0 {
			iterFunc, closeFunc = block.GetBlockOperationsBySourceAndType(api.storage, address, oType, options)
		} else {
			iterFunc, closeFunc = block.GetBlockOperationsBySource(api.storage, address, options)
		}
		for {
			t, hasNext, c := iterFunc()
			cursor = c
			if !hasNext {
				break
			}

			var blk *block.Block
			var ok bool
			if blk, ok = blockCache[t.Height]; !ok {
				if blk0, err := block.GetBlockByHeight(api.storage, t.Height); err != nil {
					break
				} else {
					blockCache[t.Height] = &blk0
					blk = &blk0
				}
			}

			r := resource.NewOperation(&t)
			r.Block = blk
			txs = append(txs, r)
		}
		closeFunc()
		return txs
	}

	if httputils.IsEventStream(r) {
		event := fmt.Sprintf("source-%s", address)
		if len(oType) > 0 {
			event = fmt.Sprintf("source-type-%s%s", address, oType)
		}
		es := NewEventStream(
			w,
			r,
			func(args ...interface{}) ([]byte, error) {
				if len(args) <= 1 {
					return nil, fmt.Errorf("render: value is empty")
				}
				i := args[1]

				if i == nil {
					return nil, nil
				}

				bo := i.(*block.BlockOperation)
				r := resource.NewOperation(bo)

				if blk, err := block.GetBlockByHeight(api.storage, bo.Height); err != nil {
					return nil, err
				} else {
					r.Block = &blk
				}

				return json.Marshal(r.Resource())
			},
			DefaultContentType,
		)

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
	list := p.ResourceList(txs, cursor)
	httputils.MustWriteJSON(w, 200, list)
}
