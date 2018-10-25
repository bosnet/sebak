package resource

import (
	"strings"

	"boscoin.io/sebak/lib/block"

	"boscoin.io/sebak/lib/transaction/operation"
	"github.com/nvellon/hal"
)

type Operation struct {
	bo *block.BlockOperation
}

func NewOperation(bo *block.BlockOperation) *Operation {
	o := &Operation{
		bo: bo,
	}
	return o
}

func (o Operation) GetMap() hal.Entry {
	body, _ := operation.UnmarshalBodyJSON(o.bo.Type, o.bo.Body)

	return hal.Entry{
		"hash":    o.bo.Hash,
		"source":  o.bo.Source,
		"type":    o.bo.Type,
		"tx_hash": o.bo.TxHash,
		"body":    body,
	}
}

func (o Operation) Resource() *hal.Resource {
	r := hal.NewResource(o, o.LinkSelf())
	r.AddNewLink("transaction", strings.Replace(URLTransactionByHash, "{id}", o.bo.TxHash, -1))
	return r
}

func (o Operation) LinkSelf() string {
	return strings.Replace(URLOperations, "{id}", o.bo.Hash, -1)
}
