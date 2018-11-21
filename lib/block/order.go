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

func (o *BlockOrder) String() string {
	var ss []string
	for _, p := range o.parts {
		ss = append(ss, strconv.FormatUint(p, 10))
	}
	return strings.Join(ss, "-")
}
