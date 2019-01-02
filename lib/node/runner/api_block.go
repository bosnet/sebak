package runner

import (
	"net/http"
	"strconv"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/errors"
	api "boscoin.io/sebak/lib/node/runner/node_api"
)

const GetBlocksPattern = "/blocks"

func (nh NetworkHandlerNode) GetBlocksHandler(w http.ResponseWriter, r *http.Request) {
	options, err := NewGetBlocksOptionsFromRequest(r)
	if err != nil {
		http.Error(w, errors.InvalidQueryString.Error(), http.StatusBadRequest)
		return
	}

	if len(options.Cursor()) > 0 {
		cursorBlock, err := block.GetBlock(nh.storage, string(options.Cursor()))
		if err != nil {
			if err == errors.StorageRecordDoesNotExist {
				http.Error(w, errors.InvalidQueryString.Error(), http.StatusBadRequest)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		options.SetCursor([]byte(cursorBlock.NewBlockKeyConfirmed()))
	}

	var bs []*block.Block
	if len(options.Hashes) > 0 {
		for _, hash := range options.Hashes {
			if exists, err := block.ExistsBlock(nh.storage, hash); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			} else if !exists {
				http.Error(w, errors.StorageRecordDoesNotExist.Error(), http.StatusNotFound)
				return
			}
		}

		for _, hash := range options.Hashes {
			b, err := block.GetBlock(nh.storage, hash)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			bs = append(bs, &b)
		}
	} else if options.Height() > 0 {
		for i := options.HeightRange[0]; i < options.HeightRange[1]; i++ {
			if options.Limit() > 0 && i-options.HeightRange[0] >= options.Limit() {
				break
			}

			if exists, err := block.ExistsBlockByHeight(nh.storage, i); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			} else if !exists {
				http.Error(w, errors.StorageRecordDoesNotExist.Error(), http.StatusNotFound)
				return
			}
		}

		for i := options.HeightRange[0]; i < options.HeightRange[1]; i++ {
			if options.Limit() > 0 && uint64(len(bs)) >= options.Limit() {
				break
			}
			b, err := block.GetBlockByHeight(nh.storage, i)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			bs = append(bs, &b)
		}
	} else {
		iterFunc, closeFunc := block.GetBlocksByConfirmed(nh.storage, options)
		for {
			b, hasNext, _ := iterFunc()
			if !hasNext {
				break
			}
			bs = append(bs, &b)
		}
		closeFunc()
	}

	w.Header().Set("Content-Type", "application/json")

	// set header; `X-SEBAK-xxx` indicates the basic explanation of the
	// response.
	w.Header().Set("X-SEBAK-RESULT-COUNT", strconv.FormatInt(int64(len(bs)), 10))

	for _, b := range bs {
		var itemType api.NodeItemDataType
		if options.Mode == GetBlocksOptionsModeHeader {
			itemType = api.NodeItemBlockHeader
			nh.renderNodeItem(w, itemType, b.Header)
		} else {
			itemType = api.NodeItemBlock
			nh.renderNodeItem(w, itemType, b)
		}

		if options.Mode == GetBlocksOptionsModeFull {
			var err error
			var tx block.BlockTransaction
			var tp block.TransactionPool

			if tx, err = block.GetBlockTransaction(nh.storage, b.ProposerTransaction); err != nil {
				nh.renderNodeItem(w, api.NodeItemError, err)
			} else if tp, err = block.GetTransactionPool(nh.storage, tx.Hash); err != nil {
				nh.renderNodeItem(w, api.NodeItemError, err)
			} else {
				tx.Message = tp.Message
				nh.renderNodeItem(w, api.NodeItemBlockTransaction, tx)
			}

			for _, t := range b.Transactions {
				if tx, err = block.GetBlockTransaction(nh.storage, t); err != nil {
					nh.renderNodeItem(w, api.NodeItemError, err)
					continue
				} else if tp, err = block.GetTransactionPool(nh.storage, tx.Hash); err != nil {
					nh.renderNodeItem(w, api.NodeItemError, err)
				} else {
					tx.Message = tp.Message
					nh.renderNodeItem(w, api.NodeItemBlockTransaction, tx)
				}
			}
		}
	}

	return
}
