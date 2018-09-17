package resource

import (
	"strings"

	"boscoin.io/sebak/lib/block"

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
	return hal.Entry{
		"hash":   o.bo.Hash,
		"source": o.bo.Source,
		"target": o.bo.Target,
		"type":   o.bo.Type,
		"amount": o.bo.Amount.String(),
	}
}

func (o Operation) Resource() *hal.Resource {
	r := hal.NewResource(o, o.LinkSelf())
	r.AddNewLink("transactions", strings.Replace(URLTransactions, "{id}", o.bo.TxHash, -1))
	return r
}

func (o Operation) LinkSelf() string {
	return strings.Replace(URLOperations, "{id}", o.bo.Hash, -1)
}
