package errors

import (
	"encoding/json"
	"io"
	"sort"

	"github.com/ethereum/go-ethereum/rlp"
)

type Error struct {
	Code    uint                   `json:"code"`
	Message string                 `json:"message"`
	Data    map[string]interface{} `json:"data" rlp:"-"`
}

func (o *Error) Serialize() (b []byte, err error) {
	b, err = json.Marshal(o)
	return
}

func (o *Error) Error() string {
	b, _ := o.Serialize()
	return string(b)
}

func (o *Error) SetData(k string, v interface{}) *Error {
	o.Data[k] = v

	return o
}

func (o *Error) Clone() *Error {
	var new Error
	new = *o

	new.Data = map[string]interface{}{}
	if o.Data != nil && len(o.Data) > 0 {
		for k, v := range o.Data {
			new.Data[k] = v
		}
	}

	return &new
}

func (o *Error) EncodeRLP(w io.Writer) (err error) {
	if o == nil {
		return rlp.Encode(w, []uint{})
	}

	if o.Data != nil && len(o.Data) > 0 {
		var d [][2]interface{}

		var keys []string
		for k, _ := range o.Data {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			d = append(d, [2]interface{}{k, o.Data[k]})
		}
		err = rlp.Encode(w, d)
	}

	return rlp.Encode(w, struct {
		Code    uint
		Message string
	}{
		Code:    o.Code,
		Message: o.Message,
	})
}

func NewError(code uint, message string) *Error {
	return &Error{Code: code, Message: message, Data: map[string]interface{}{}}
}
