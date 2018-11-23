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

	var blk *block.Block
	if blk, err = api.getBlockByTxHash(hash); err != nil {
		httputils.WriteJSONError(w, err)
		return
	}

	ops, cursor := api.getOperationsByTxHash(hash, blk, options)
	if len(ops) < 1 {
		httputils.WriteJSONError(w, errors.BlockTransactionDoesNotExists)
		return
	}

	list := p.ResourceList(ops, cursor)
	httputils.MustWriteJSON(w, 200, list)
}

func (api NetworkHandlerAPI) getOperationsByTxHash(txHash string, blk *block.Block, options storage.ListOptions) (txs []resource.Resource, cursor []byte) {
	iterFunc, closeFunc := block.GetBlockOperationsByTxHash(api.storage, txHash, options)
	for {
		o, hasNext, c := iterFunc()
		cursor = c
		if !hasNext {
			break
		}
		rs := resource.NewOperation(&o)
		rs.Block = blk
		txs = append(txs, rs)
	}
	closeFunc()
	return
}

func (api NetworkHandlerAPI) getBlockByTxHash(hash string) (*block.Block, error) {
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
