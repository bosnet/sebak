package sebakerror

import "encoding/json"

type Error struct {
	Code    uint   `json:"code"`
	Message string `json:"message"`
}

func (o Error) Serialize() (b []byte, err error) {
	b, err = json.Marshal(o)
	return
}

func (o Error) Error() string {
	b, _ := o.Serialize()
	return string(b)
}

func NewError(code uint, message string) Error {
	return Error{Code: code, Message: message}
}
