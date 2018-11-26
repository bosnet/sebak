package api

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/errors"
	"boscoin.io/sebak/lib/node/runner/api/resource"
	"boscoin.io/sebak/lib/storage"
)

const (
	DefaultLimit uint64 = 20
	MaxLimit     uint64 = 100
)

type PageQuery struct {
	request *http.Request
	cursor  []byte
	reverse bool
	limit   uint64
}

func NewPageQuery(r *http.Request) (*PageQuery, error) {
	p := &PageQuery{
		request: r,
		limit:   DefaultLimit,
	}
	err := p.parseRequest()
	return p, err
}

func (p *PageQuery) Limit() uint64 {
	return p.limit
}

func (p *PageQuery) Reverse() bool {
	return p.reverse
}

func (p *PageQuery) Cursor() []byte {
	return p.cursor
}

func (p *PageQuery) SelfLink() string {
	return p.request.URL.String()
}

func (p *PageQuery) PrevLink(cursor []byte) string {
	path := p.request.URL.Path
	query := p.urlValues(cursor, true, p.limit).Encode()
	link := fmt.Sprintf("%s?%s", path, query)
	return link
}

func (p *PageQuery) NextLink(cursor []byte) string {
	path := p.request.URL.Path
	query := p.urlValues(cursor, false, p.limit).Encode()
	link := fmt.Sprintf("%s?%s", path, query)
	return link
}

func (p *PageQuery) ListOptions() storage.ListOptions {
	return storage.NewDefaultListOptions(p.Reverse(), p.Cursor(), p.Limit())
}

func (p *PageQuery) WalkOption() *storage.WalkOption {
	return storage.NewWalkOption(string(p.Cursor()), p.Limit(), p.Reverse(), true)
}

func (p *PageQuery) ResourceList(rs []resource.Resource, firstCursor, lastCursor []byte) *resource.ResourceList {
	if p.reverse {
		return resource.NewResourceList(rs, p.SelfLink(), p.NextLink(firstCursor), p.PrevLink(lastCursor))
	} else {
		return resource.NewResourceList(rs, p.SelfLink(), p.NextLink(lastCursor), p.PrevLink(firstCursor))
	}
}

func (p *PageQuery) parseRequest() error {
	q := p.request.URL.Query()
	r := q.Get("reverse")
	if r != "" {
		reverse, err := common.ParseBoolQueryString(r)
		if err != nil {
			return err
		}
		p.reverse = reverse
	}
	c := q.Get("cursor")
	if c != "" {
		var cursorByte []byte
		var err error
		if cursorByte, err = base64.StdEncoding.DecodeString(c); err != nil {
			p.cursor = []byte(c)
		} else {
			p.cursor = cursorByte
		}
	}

	l := q.Get("limit")
	if l != "" {
		limit, err := strconv.ParseUint(l, 10, 64)
		if err != nil {
			return err
		}

		if limit > MaxLimit {
			return errors.PageQueryLimitMaxExceed
		}
		p.limit = limit
	}
	return nil
}

func (p PageQuery) urlValues(cursor []byte, reverse bool, limit uint64) url.Values {
	v := url.Values{
		"reverse": []string{strconv.FormatBool(reverse)},
	}

	if len(cursor) > 0 {
		v.Set("cursor", string(base64.StdEncoding.EncodeToString(cursor)))
	}
	if p.limit > 0 {
		v.Set("limit", strconv.FormatUint(p.limit, 10))
	}

	return v
}
