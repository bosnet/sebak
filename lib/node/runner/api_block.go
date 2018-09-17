package runner

import (
	"bufio"
	"bytes"
	"encoding/json"
	"net/http"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/error"
)

const GetBlocksPattern = "/blocks"

type GetBlocksDataType string

const (
	GetBlocksDataTypeBlock       GetBlocksDataType = "block"
	GetBlocksDataTypeHeader      GetBlocksDataType = "header"
	GetBlocksDataTypeTransaction GetBlocksDataType = "transaction"
	GetBlocksDataTypeOperation   GetBlocksDataType = "operation"
	GetBlocksDataTypeError       GetBlocksDataType = "error"
)

func (nh NetworkHandlerNode) GetBlocksHandler(w http.ResponseWriter, r *http.Request) {
	options, err := NewGetBlocksOptionsFromRequest(r)
	if err != nil {
		http.Error(w, errors.ErrorInvalidQueryString.Error(), http.StatusBadRequest)
		return
	}

	if len(options.Cursor()) > 0 {
		cursorBlock, err := block.GetBlock(nh.storage, string(options.Cursor()))
		if err != nil {
			if err == errors.ErrorStorageRecordDoesNotExist {
				http.Error(w, errors.ErrorInvalidQueryString.Error(), http.StatusBadRequest)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		options.SetCursor([]byte(cursorBlock.NewBlockKeyConfirmed()))
	}

	var bs []interface{}
	if len(options.Hashes) > 0 {
		for _, hash := range options.Hashes {
			if exists, err := block.ExistsBlock(nh.storage, hash); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			} else if !exists {
				http.Error(w, errors.ErrorStorageRecordDoesNotExist.Error(), http.StatusNotFound)
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
				http.Error(w, errors.ErrorStorageRecordDoesNotExist.Error(), http.StatusNotFound)
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
		var itemType GetBlocksDataType
		if options.Mode == GetBlocksOptionsModeHeader {
			itemType = GetBlocksDataTypeHeader
			renderGetBlocksItem(w, itemType, b.(*block.Block).Header)
		} else {
			itemType = GetBlocksDataTypeBlock
			renderGetBlocksItem(w, itemType, b.(*block.Block))
		}

		if options.Mode == GetBlocksOptionsModeFull {
			var err error
			var tx block.BlockTransaction
			var op block.BlockOperation

			bk := b.(*block.Block)
			for _, t := range bk.Transactions {
				if tx, err = block.GetBlockTransaction(nh.storage, t); err != nil {
					renderGetBlocksItem(w, GetBlocksDataTypeError, err)
					continue
				}
				renderGetBlocksItem(w, GetBlocksDataTypeTransaction, tx)

				for _, opHash := range tx.Operations {
					if op, err = block.GetBlockOperation(nh.storage, opHash); err != nil {
						renderGetBlocksItem(w, GetBlocksDataTypeError, err)
						continue
					}
					renderGetBlocksItem(w, GetBlocksDataTypeOperation, op)
				}
			}
		}
	}

	return
}

func renderGetBlocksItem(w http.ResponseWriter, itemType GetBlocksDataType, o interface{}) {
	s, err := json.Marshal(o)
	if err != nil {
		itemType = GetBlocksDataTypeError
		s = []byte(err.Error())
	}

	w.Write(append([]byte(itemType+" "), append(s, '\n')...))
}

func UnmarshalGetBlocksHandlerItem(d []byte) (itemType GetBlocksDataType, b interface{}, err error) {
	sc := bufio.NewScanner(bytes.NewReader(d))
	sc.Split(bufio.ScanWords)
	sc.Scan()

	unmarshal := func(o interface{}) error {
		if err := json.Unmarshal(d[len(sc.Bytes())+1:], o); err != nil {
			return err
		}
		return nil
	}

	itemType = GetBlocksDataType(sc.Text())
	switch itemType {
	case GetBlocksDataTypeBlock:
		var t block.Block
		err = unmarshal(&t)
		b = t
	case GetBlocksDataTypeHeader:
		var t block.Header
		err = unmarshal(&t)
		b = t
	case GetBlocksDataTypeTransaction:
		var t block.BlockTransaction
		err = unmarshal(&t)
		b = t
	case GetBlocksDataTypeOperation:
		var t block.BlockOperation
		err = unmarshal(&t)
		b = t
	case GetBlocksDataTypeError:
		var t errors.Error
		err = unmarshal(&t)
		b = t
	default:
		err = errors.ErrorInvalidMessage
	}

	return
}
