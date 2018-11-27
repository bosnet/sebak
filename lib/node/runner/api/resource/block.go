package resource

import (
	"boscoin.io/sebak/lib/common"
	"strings"

	"boscoin.io/sebak/lib/block"
	"github.com/nvellon/hal"
)

type Block struct {
	b *block.Block
}

func NewBlock(b *block.Block) *Block {
	blk := &Block{
		b: b,
	}
	return blk
}

func (blk Block) GetMap() hal.Entry {
	b := blk.b
	return hal.Entry{
		"version":              b.Version,
		"hash":                 b.Hash,
		"height":               b.Height,
		"prev_block_hash":      b.PrevBlockHash,
		"transactions_root":    b.TransactionsRoot,
		"confirmed":            b.Confirmed,
		"proposer":             b.Proposer,
		"proposed_time":        b.ProposedTime,
		"proposer_transaction": b.ProposerTransaction,
		"round":                b.Round,
		"transactions":         b.Transactions,
	}
}

func (blk Block) Resource() *hal.Resource {
	r := hal.NewResource(blk, blk.LinkSelf())
	return r
}

func (blk Block) LinkSelf() string {
	return strings.Replace(URLBlocks, "{id}", blk.b.Hash, -1)
}

func (blk Block) MarshalJSON() ([]byte, error) {
	r := blk.Resource()
	return common.JSONMarshalWithoutEscapeHTML(r.GetMap())
}
