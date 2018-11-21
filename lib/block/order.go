package block

import (
	"strconv"
	"strings"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/storage"
)

type BlockOrder struct {
	parts []uint64 // height,txindex,opindex
}

func NewBlockOrder(height uint64) *BlockOrder {
	b := &BlockOrder{
		parts: []uint64{height},
	}
	return b
}

func NewBlockTxOrder(height, index uint64) *BlockOrder {
	b := &BlockOrder{
		parts: []uint64{height, index},
	}
	return b
}

func NewBlockOpOrder(height, txindex, index uint64) *BlockOrder {
	b := &BlockOrder{
		parts: []uint64{height, txindex, index},
	}
	return b
}

func (o *BlockOrder) Index(idx *storage.Index) string {
	for _, x := range o.parts {
		idx.WriteOrder(common.EncodeUint64ToString(x))
	}
	return idx.String()
}

func (o *BlockOrder) formatText(xs []uint64) string {
	var ss []string
	for _, p := range xs {
		ss = append(ss, strconv.FormatUint(p, 10))
	}
	return strings.Join(ss, "-")
}

func (o *BlockOrder) String() string {
	return o.formatText(o.parts)
}

func (o *BlockOrder) NextString() string {
	return o.formatText(o.NextOrder())
}

func (o *BlockOrder) PrevString() string {
	return o.formatText(o.PrevOrder())
}

func (o *BlockOrder) NextOrder() (xs []uint64) {
	if o == nil || len(o.parts) <= 0 {
		return
	}
	for i, x := range o.parts {
		if i == len(xs)-1 {
			x++
		}
		xs = append(xs, x)
	}
	return
}

func (o *BlockOrder) PrevOrder() []uint64 {
	if o == nil || len(o.parts) <= 0 {
		return []uint64{}
	}
	return blockOrderPrevOrder(o.parts, len(o.parts)-1)
}

func blockOrderPrevOrder(xs []uint64, pos int) []uint64 {
	if pos == 0 || len(xs)-1 < pos {
		return xs
	}
	if xs[pos] > 0 {
		xs[pos]--
		return xs
	}
	return blockOrderPrevOrder(xs, pos-1)
}
