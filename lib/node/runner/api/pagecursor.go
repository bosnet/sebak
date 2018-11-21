package api

import (
	"strconv"
	"strings"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/storage"
)

type PageCursor struct {
	input  string
	prefix string
	index  *storage.Index
}

func NewPageCursor(input []byte, prefix string) *PageCursor {
	p := &PageCursor{
		input:  strings.TrimSpace(string(input)),
		prefix: prefix,
		index:  storage.NewIndex(),
	}

	return p
}

func (p *PageCursor) IndexKey() ([]byte, error) {
	p.index.WritePrefix(p.prefix)
	err := p.indexOrder()
	return p.index.Bytes(), err
}

func (p *PageCursor) indexOrder() error {
	parts := strings.Split(p.input, "-")
	for i, part := range parts {
		if part == "" {
			continue
		}
		if i < 3 {
			partInt, err := strconv.ParseUint(part, 10, 64)
			if err != nil {
				return err
			}
			p.index.WriteOrder(common.EncodeUint64ToString(partInt))
		}
	}
	return nil
}
