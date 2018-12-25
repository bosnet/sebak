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

	options, err := p.PageCursorListOptions(block.GetBlockOperationKeyPrefixTxHash(hash))
	if err != nil {
		httputils.WriteJSONError(w, err)
		return
	}

	var blk *block.Block
	if blk, err = api.getBlockByTx(hash); err != nil {
		httputils.WriteJSONError(w, err)
		return
	}

	ops, pOrder, nOrder := api.getOperationsByTx(hash, blk, options)

	if len(ops) < 1 {
		httputils.WriteJSONError(w, errors.BlockTransactionDoesNotExists)
		return
	}

	list := p.ResourceListWithOrder(ops, pOrder, nOrder)
	httputils.MustWriteJSON(w, 200, list)
}

func (api NetworkHandlerAPI) getOperationsByTx(txHash string, blk *block.Block, options storage.ListOptions) (txs []resource.Resource, pOrder *block.BlockOrder, nOrder *block.BlockOrder) {
	iterFunc, closeFunc := block.GetBlockOperationsByTx(api.storage, txHash, options)
	for idx := 0; ; idx++ {
		o, hasNext, _ := iterFunc()
		if !hasNext {
			break
		}
		if pOrder == nil {
			pOrder = o.BlockOrder()
		}
		nOrder = o.BlockOrder()

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
