package resource

import (
	"strings"

	"boscoin.io/sebak/lib/block"
	"github.com/nvellon/hal"
)

type Transaction struct {
	bt *block.BlockTransaction
}

func NewTransaction(bt *block.BlockTransaction) *Transaction {
	t := &Transaction{
		bt: bt,
	}
	return t
}

func (t Transaction) GetMap() hal.Entry {
	return hal.Entry{
		"hash":            t.bt.Hash,
		"block":           t.bt.Block,
		"source":          t.bt.Source,
		"fee":             t.bt.Fee.String(),
		"sequence_id":     t.bt.SequenceID,
		"created":         t.bt.Created,
		"operation_count": len(t.bt.Operations),
		"index":           t.bt.Index,
	}
}
func (t Transaction) Resource() *hal.Resource {

	r := hal.NewResource(t, t.LinkSelf())
	r.AddLink("account", hal.NewLink(strings.Replace(URLAccounts, "{id}", t.bt.Source, -1)))
	r.AddLink("operations", hal.NewLink(strings.Replace(URLTransactionOperations, "{id}", t.bt.Hash, -1)+"{?cursor,limit,order}", hal.LinkAttr{"templated": true}))
	return r
}

func (t Transaction) LinkSelf() string {
	return strings.Replace(URLTransactions, "{id}", t.bt.Hash, -1)
}

type TransactionStatus struct {
	Hash   string
	Status string
}

func NewTransactionStatus(hash, status string) *TransactionStatus {
	t := &TransactionStatus{
		Hash:   hash,
		Status: status,
	}
	return t
}

func (t TransactionStatus) GetMap() hal.Entry {
	return hal.Entry{
		"hash":   t.Hash,
		"status": t.Status,
	}
}
func (t TransactionStatus) Resource() *hal.Resource {

	r := hal.NewResource(t, t.LinkSelf())
	r.AddLink("transaction", hal.NewLink(strings.Replace(URLTransactionByHash, "{id}", t.Hash, -1)))
	return r
}

func (t TransactionStatus) LinkSelf() string {
	return strings.Replace(URLTransactionStatus, "{id}", t.Hash, -1)
}
