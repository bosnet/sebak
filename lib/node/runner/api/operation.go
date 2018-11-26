package api

import (
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/errors"
	"boscoin.io/sebak/lib/network/httputils"
	"boscoin.io/sebak/lib/node/runner/api/resource"
	"boscoin.io/sebak/lib/transaction/operation"
)

func (api NetworkHandlerAPI) GetOperationsByTxHashOpIndexHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	txHash := vars["id"]
	opIndex := vars["opindex"]

	opIndexInt, err := strconv.Atoi(opIndex)
	if err != nil {
		httputils.WriteJSONError(w, err)
		return
	}

	readFunc := func() (payload interface{}, err error) {
		found, err := block.ExistsBlockTransaction(api.storage, txHash)
		if err != nil {
			return nil, err
		}
		if !found {
			return nil, errors.BlockTransactionDoesNotExists
		}
		bo, err := block.GetBlockOperationWithIndex(api.storage, txHash, opIndexInt)
		if err != nil {
			return nil, err
		}
		payload = resource.NewOperation(&bo, opIndexInt)
		return payload, nil
	}

	payload, err := readFunc()
	if err != nil {
		httputils.WriteJSONError(w, err)
		return
	}

	httputils.MustWriteJSON(w, 200, payload)
}

func (api NetworkHandlerAPI) GetOperationsByAccountHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	address := vars["id"]

	p, err := NewPageQuery(r)
	if err != nil {
		httputils.WriteJSONError(w, err)
		return
	}

	options := p.ListOptions()

	var oType operation.OperationType
	oTypeStr := r.URL.Query().Get("type")
	if len(oTypeStr) > 0 {
		if err = oType.UnmarshalText([]byte(oTypeStr)); err != nil {
			httputils.WriteJSONError(w, errors.InvalidQueryString)
			return
		}
	}

	if found, err := block.ExistsBlockAccount(api.storage, address); err != nil {
		httputils.WriteJSONError(w, err)
		return
	} else if !found {
		httputils.WriteJSONError(w, errors.BlockAccountDoesNotExists)
		return
	}

	var txs []resource.Resource
	blockCache := map[ /* block.Height */ uint64]*block.Block{}
	var firstCursor []byte
	var lastCursor []byte
	{

		var iterFunc func() (block.BlockOperation, bool, []byte)
		var closeFunc func()
		if len(oTypeStr) > 0 {
			iterFunc, closeFunc = block.GetBlockOperationsByPeersAndType(api.storage, address, oType, options)
		} else {
			iterFunc, closeFunc = block.GetBlockOperationsByPeers(api.storage, address, options)
		}
		for {
			t, hasNext, c := iterFunc()
			if !hasNext {
				break
			}
			if len(firstCursor) == 0 {
				firstCursor = append(firstCursor, c...)
			}
			lastCursor = append([]byte{}, c...)

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
			//GetOperationIndex
			bt, err := block.GetBlockTransaction(api.storage, t.TxHash)
			if err != nil {
				httputils.WriteJSONError(w, err)
				return
			}
			opIndex, err := bt.GetOperationIndex(t.Hash)
			if err != nil {
				httputils.WriteJSONError(w, err)
				return
			}
			r := resource.NewOperation(&t, opIndex)
			r.Block = blk
			txs = append(txs, r)
		}
		closeFunc()
	}

	list := p.ResourceList(txs, firstCursor, lastCursor)
	httputils.MustWriteJSON(w, 200, list)
}
