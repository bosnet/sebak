package resource

import (
	"fmt"
	"strings"

	"boscoin.io/sebak/lib/block"

	"github.com/nvellon/hal"

	"boscoin.io/sebak/lib/transaction/operation"
)

type Operation struct {
	Block   *block.Block
	bo      *block.BlockOperation
	opIndex int
}

func NewOperation(bo *block.BlockOperation, opIndex int) *Operation {
	o := &Operation{
		bo:      bo,
		opIndex: opIndex,
	}
	return o
}

func (o Operation) BlockOperation() *block.BlockOperation {
	return o.bo
}

func (o Operation) GetMap() hal.Entry {
	body, _ := operation.UnmarshalBodyJSON(o.bo.Type, o.bo.Body)

	entry := hal.Entry{
		"hash":         o.bo.Hash,
		"source":       o.bo.Source,
		"target":       o.bo.Target,
		"type":         o.bo.Type,
		"tx_hash":      o.bo.TxHash,
		"index":        o.bo.Index,
		"body":         body,
		"block_height": o.bo.Height,
	}

	if o.Block != nil {
		entry["confirmed"] = o.Block.Confirmed
		entry["proposed_time"] = o.Block.ProposedTime
	}

	return entry
}

func (o Operation) Resource() *hal.Resource {
	r := hal.NewResource(o, o.LinkSelf())
	r.AddNewLink("transaction", strings.Replace(URLTransactionByHash, "{id}", o.bo.TxHash, -1))
	return r
}

func (o Operation) LinkSelf() string {
	self := strings.Replace(URLTransactionOperation, "{id}", o.bo.TxHash, -1)
	self = strings.Replace(self, "{opindex}", fmt.Sprintf("%d", o.opIndex), -1)
	return self
}
