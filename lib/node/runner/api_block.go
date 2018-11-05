package runner

import (
	"bufio"
	"bytes"
	"encoding/json"
	"net/http"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/errors"
	"boscoin.io/sebak/lib/transaction"
)

const GetBlocksPattern = "/blocks"

type NodeItemDataType string

const (
	NodeItemBlock            NodeItemDataType = "block"
	NodeItemBlockHeader      NodeItemDataType = "block-header"
	NodeItemBlockTransaction NodeItemDataType = "block-transaction"
	NodeItemTransaction      NodeItemDataType = "transaction"
	NodeItemError            NodeItemDataType = "error"
)

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
	w.Header().Set("X-SEBAK-RESULT-COUNT", string(len(bs)))

	for _, b := range bs {
		var itemType NodeItemDataType
		if options.Mode == GetBlocksOptionsModeHeader {
			itemType = NodeItemBlockHeader
			nh.renderNodeItem(w, itemType, b.Header)
		} else {
			itemType = NodeItemBlock
			nh.renderNodeItem(w, itemType, b)
		}

		if options.Mode == GetBlocksOptionsModeFull {
			var err error
			var tx block.BlockTransaction
			var tp block.TransactionPool

			if tx, err = block.GetBlockTransaction(nh.storage, b.ProposerTransaction); err != nil {
				nh.renderNodeItem(w, NodeItemError, err)
			} else if tp, err = block.GetTransactionPool(nh.storage, tx.Hash); err != nil {
				nh.renderNodeItem(w, NodeItemError, err)
			} else {
				tx.Message = tp.Message
				nh.renderNodeItem(w, NodeItemBlockTransaction, tx)
			}

			for _, t := range b.Transactions {
				if tx, err = block.GetBlockTransaction(nh.storage, t); err != nil {
					nh.renderNodeItem(w, NodeItemError, err)
					continue
				} else if tp, err = block.GetTransactionPool(nh.storage, tx.Hash); err != nil {
					nh.renderNodeItem(w, NodeItemError, err)
				} else {
					tx.Message = tp.Message
					nh.renderNodeItem(w, NodeItemBlockTransaction, tx)
				}
			}
		}
	}

	return
}

func UnmarshalNodeItemResponse(d []byte) (itemType NodeItemDataType, b interface{}, err error) {
	sc := bufio.NewScanner(bytes.NewReader(d))
	sc.Split(bufio.ScanWords)
	sc.Scan()
	if err = sc.Err(); err != nil {
		return
	}

	unmarshal := func(o interface{}) error {
		if err := json.Unmarshal(d[len(sc.Bytes())+1:], o); err != nil {
			return err
		}
		return nil
	}

	itemType = NodeItemDataType(sc.Text())
	switch itemType {
	case NodeItemBlock:
		var t block.Block
		err = unmarshal(&t)
		b = t
	case NodeItemBlockHeader:
		var t block.Header
		err = unmarshal(&t)
		b = t
	case NodeItemBlockTransaction:
		var t block.BlockTransaction
		err = unmarshal(&t)
		b = t
	case NodeItemTransaction:
		var t transaction.Transaction
		err = unmarshal(&t)
		b = t
	case NodeItemError:
		var t errors.Error
		err = unmarshal(&t)
		b = &t
	default:
		err = errors.InvalidMessage
	}

	return
}
