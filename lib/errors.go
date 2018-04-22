package sebak

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

// pre-defined `Errors`
var ErrorBlockAlreayExists = Error{Code: 100, Message: "already exists in block"}
