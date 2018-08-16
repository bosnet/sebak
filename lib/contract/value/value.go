package value

import (
	"encoding/binary"
	"io"

	"github.com/robertkrimen/otto"
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
	if ottoValue.IsNumber() {
		intValue, err := ottoValue.ToInteger()
		if err != nil {
			return nil, err
		}

		v.Type = Int
		v.Contents = make([]byte, 8)
		binary.LittleEndian.PutUint64(v.Contents, uint64(intValue))

	} else if ottoValue.IsString() {
		strValue, err := ottoValue.ToString()
		if err != nil {
			return nil, err
		}

		v.Type = String
		v.Contents = []byte(strValue)

	} else {
		panic("value.ToValue: not yet supported type")
	}
	return v, nil
}
