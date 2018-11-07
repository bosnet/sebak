package api

import (
	"net/http"

	"github.com/gorilla/mux"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/errors"
	"boscoin.io/sebak/lib/network/httputils"
	"boscoin.io/sebak/lib/node/runner/api/resource"
	"boscoin.io/sebak/lib/storage"
)

func (api NetworkHandlerAPI) GetOperationsByTxHandler(w http.ResponseWriter, r *http.Request) {
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

	var blk *block.Block
	if blk, err = api.getBlockByTx(hash); err != nil {
		httputils.WriteJSONError(w, err)
		return
	}

	ops, firstCursor, cursor := api.getOperationsByTx(hash, blk, options)
	if len(ops) < 1 {
		httputils.WriteJSONError(w, errors.BlockTransactionDoesNotExists)
		return
	}

	list := p.ResourceList(ops, firstCursor, cursor)
	httputils.MustWriteJSON(w, 200, list)
}

func (api NetworkHandlerAPI) getOperationsByTx(txHash string, blk *block.Block, options storage.ListOptions) (txs []resource.Resource, firstCursor, cursor []byte) {
	iterFunc, closeFunc := block.GetBlockOperationsByTx(api.storage, txHash, options)
	for idx := 0; ; idx++ {
		o, hasNext, c := iterFunc()
		if !hasNext {
			break
		}
		cursor = append([]byte{}, c...)
		if len(firstCursor) == 0 {
			firstCursor = append(firstCursor, c...)
		}

		rs := resource.NewOperation(&o, idx)
		rs.Block = blk
		txs = append(txs, rs)
	}
	closeFunc()
	return
}

func (api NetworkHandlerAPI) getBlockByTx(hash string) (*block.Block, error) {
	// get block by it's `Height`
	if found, err := block.ExistsBlockTransaction(api.storage, hash); err != nil {
		return nil, err
	} else if !found {
		return nil, errors.BlockTransactionDoesNotExists.Clone().SetData("status", http.StatusNotFound)
	}

	var bt block.BlockTransaction
	var err error
	if bt, err = block.GetBlockTransaction(api.storage, hash); err != nil {
		return nil, err
	}

	blk, err := block.GetBlock(api.storage, bt.Block)
	return &blk, err
}
