package httputils

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"boscoin.io/sebak/lib/common"
)

const DefaultMaxLimit uint64 = 100

type Paginator struct {
	request *http.Request
	cursor  []byte
	reverse bool
	limit   uint64
	err     error
}

func NewPaginator(r *http.Request) *Paginator {
	p := &Paginator{
		request: r,
		limit:   DefaultMaxLimit,
	}
	p.parseRequest()
	return p
}

func (p *Paginator) Limit() uint64 {
	return p.limit
}

func (p *Paginator) Reverse() bool {
	return p.reverse
}

func (p *Paginator) Cursor() []byte {
	return p.cursor
}

func (p *Paginator) Error() error {
	return p.err
}

func (p *Paginator) SelfLink() string {
	return p.request.URL.String()
}

func (p *Paginator) PrevLink(cursor []byte) string {
	path := p.request.URL.Path
	query := p.urlValues(cursor, false).Encode()
	link := fmt.Sprintf("%s?%s", path, query)
	return link
}

func (p *Paginator) NextLink(cursor []byte) string {
	path := p.request.URL.Path
	query := p.urlValues(cursor, true).Encode()
	link := fmt.Sprintf("%s?%s", path, query)
	return link
}

func (p *Paginator) parseRequest() {
	q := p.request.URL.Query()
	r := q.Get("reverse")
	if r != "" {
		reverse, err := common.ParseBoolQueryString(r)
		if err != nil {
			p.err = err
			return
		}
		p.reverse = reverse
	}
	c := q.Get("cursor")
	if c != "" {
		p.cursor = []byte(c)
	}

	l := q.Get("limit")
	if l != "" {
		limit, err := strconv.ParseUint(l, 10, 64)
		if err != nil {
			p.err = err
			return
		}
		p.limit = limit
	}
}

func (p Paginator) urlValues(cursor []byte, reverse bool) url.Values {
	v := url.Values{
		"reverse": []string{strconv.FormatBool(reverse)},
	}

	if len(cursor) > 0 {
		v.Set("cursor", string(p.cursor))
	}
	if p.limit > 0 {
		v.Set("limit", strconv.FormatUint(p.limit, 10))
	}

	return v
}
