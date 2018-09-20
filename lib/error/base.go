package errors

import "encoding/json"

type Error struct {
	Code    uint   `json:"code"`
	Message string `json:"message"`
	Data    map[string]interface{}
}

func (o *Error) Serialize() (b []byte, err error) {
	b, err = json.Marshal(o)
	return
}

func (o *Error) Error() string {
	b, _ := o.Serialize()
	return string(b)
}

// SetData sets `Error.Data`
func (o *Error) SetData(k string, v interface{}) *Error {
	o.Data[k] = v

	return o
}

func (o *Error) Clone() *Error {
	data := map[string]interface{}{}
	if o.Data != nil {
		for k, v := range o.Data {
			data[k] = v
		}
	}

	return &Error{
		Code:    o.Code,
		Message: o.Message,
		Data:    data,
	}
}

func NewError(code uint, message string) *Error {
	return &Error{Code: code, Message: message, Data: map[string]interface{}{}}
}
