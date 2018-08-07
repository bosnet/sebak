package value

import (
	"encoding/binary"
	"github.com/robertkrimen/otto"
	"io"
)

type Type int

const (
	Nil Type = iota
	Int
	String
	Array
	Bytes
)

type Value struct {
	Type     Type
	Contents []byte
}

func (v *Value) Serialize(w io.Writer) error {

	return nil
}

func (v *Value) Deserialize(r io.Reader) error {

	return nil
}

func StringValue(s string) *Value {
	v := &Value{
		Type:     String,
		Contents: []byte(s),
	}
	return v
}

func ToValue(ottoValue otto.Value) (*Value, error) {
	//ottoValue.Export()
	var v = new(Value)
	var err error
	if ottoValue.IsNumber() {
		var intValue int64
		intValue, err = ottoValue.ToInteger()
		v.Type = Int
		v.Contents = make([]byte, 4)
		binary.LittleEndian.PutUint64(v.Contents, uint64(intValue))
	} else if ottoValue.IsString() {
		var strValue string
		strValue, err = ottoValue.ToString()
		v.Type = String
		v.Contents = []byte(strValue)
	} else {
		panic("not yet supported type")
	}
	return v, err
}
