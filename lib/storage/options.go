package storage

import (
	"net/url"
	"strconv"
)

var DefaultMaxLimitListOptions uint64 = 100

type ListOptions interface {
	Reverse() bool
	SetReverse(bool) ListOptions
	Cursor() []byte
	SetCursor([]byte) ListOptions
	Limit() uint64
	SetLimit(uint64) ListOptions
	Template() string
	URLValues() url.Values
	Encode() string
}

type DefaultListOptions struct {
	reverse bool
	cursor  []byte
	limit   uint64
}

func NewDefaultListOptions(reverse bool, cursor []byte, limit uint64) *DefaultListOptions {
	return &DefaultListOptions{
		reverse: reverse,
		cursor:  cursor,
		limit:   limit,
	}
}

func (o DefaultListOptions) Reverse() bool {
	return o.reverse
}

func (o *DefaultListOptions) SetReverse(r bool) ListOptions {
	o.reverse = r
	return o
}

func (o DefaultListOptions) Cursor() []byte {
	return o.cursor
}

func (o *DefaultListOptions) SetCursor(c []byte) ListOptions {
	o.cursor = c
	return o
}

func (o DefaultListOptions) Limit() uint64 {
	return o.limit
}

func (o *DefaultListOptions) SetLimit(l uint64) ListOptions {
	o.limit = l
	return o
}

func (o DefaultListOptions) Template() string {
	return "{?cursor,limit,reverse}"
}

func (o DefaultListOptions) URLValues() url.Values {
	v := url.Values{
		"reverse": []string{strconv.FormatBool(o.reverse)},
	}

	if len(o.cursor) > 0 {
		v.Set("cursor", string(o.cursor))
	}
	if o.limit > 0 {
		v.Set("limit", strconv.FormatUint(o.limit, 10))
	}

	return v
}

func (o DefaultListOptions) Encode() string {
	return o.URLValues().Encode()
}
