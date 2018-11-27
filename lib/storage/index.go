package storage

import (
	"strings"
)

const (
	IndexPrefixOrderDelimiter = "/"
	IndexElementDelimiter     = "-"
)

type Index struct {
	prefix []string
	order  []string
}

func NewIndex() *Index {
	idx := &Index{
		prefix: make([]string, 0, 0),
		order:  make([]string, 0, 0),
	}
	return idx
}

func (idx Index) Bytes() []byte {
	return []byte(idx.String())
}

func (idx Index) String() string {
	prefix := strings.Join(idx.prefix, IndexElementDelimiter)
	order := strings.Join(idx.order, IndexElementDelimiter)
	var strs []string
	if len(prefix) != 0 {
		strs = append(strs, prefix)
	}
	if len(order) != 0 {
		strs = append(strs, order)
	}

	index := strings.Join(strs, IndexPrefixOrderDelimiter)
	return index
}

func (idx *Index) WritePrefix(ss ...string) *Index {
	for _, s := range ss {
		idx.prefix = append(idx.prefix, s)
	}
	return idx
}

func (idx *Index) WriteOrder(ss ...string) *Index {
	for _, s := range ss {
		idx.order = append(idx.order, s)
	}
	return idx
}
