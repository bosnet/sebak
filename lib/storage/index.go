package storage

import "strings"

const (
	IndexPrefixOrderDelimiter = "/"
	IndexElementDelimiter     = "-"
)

type Index struct {
	prefix string
	order  string
}

func NewIndex() *Index {
	idx := &Index{}
	return idx
}

func (idx Index) Bytes() []byte {
	return []byte(idx.String())
}

func (idx Index) String() string {
	index := strings.Join([]string{idx.prefix, idx.order}, IndexPrefixOrderDelimiter)
	return index
}

func (idx *Index) WritePrefix(ss ...string) *Index {
	idx.prefix = strings.Join(ss, IndexElementDelimiter)
	return idx
}

func (idx *Index) WriteOrder(ss ...string) *Index {
	idx.order = strings.Join(ss, IndexElementDelimiter)
	return idx
}
