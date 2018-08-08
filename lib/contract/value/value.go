package value

import (
	"encoding/binary"
	"errors"
	"github.com/robertkrimen/otto"
)

type Type int

const (
	Nil Type = iota
	SInt
	UInt
	String
	Boolean
)

const (
	True  = 0x01
	False = 0x00
)

type Value struct {
	Type     Type
	Contents []byte
}

func StringValue(s string) *Value {
	var v = new(Value)
	v.Encode(s)
	return v
}

func (v *Value) Encode(iv interface{}) (err error) {
	switch iv.(type) {
	case nil:
		v.Type = Nil
		v.Contents = []byte{}
	case string:
		v.Type = String
		v.Contents = []byte(iv.(string))
	case bool:
		v.Type = Boolean
		if iv.(bool) {
			v.Contents = []byte{True}
		} else {
			v.Contents = []byte{False}
		}
	case int:
		v.Type = SInt
		v.Contents = make([]byte, 8)
		binary.LittleEndian.PutUint64(v.Contents, uint64(iv.(int)))
	case int8:
		v.Type = SInt
		v.Contents = make([]byte, 8)
		binary.LittleEndian.PutUint64(v.Contents, uint64(iv.(int8)))
	case int16:
		v.Type = SInt
		v.Contents = make([]byte, 8)
		binary.LittleEndian.PutUint64(v.Contents, uint64(iv.(int16)))
	case int32:
		v.Type = SInt
		v.Contents = make([]byte, 8)
		binary.LittleEndian.PutUint64(v.Contents, uint64(iv.(int32)))
	case int64:
		v.Type = SInt
		v.Contents = make([]byte, 8)
		binary.LittleEndian.PutUint64(v.Contents, uint64(iv.(int64)))
	case uint:
		v.Type = UInt
		v.Contents = make([]byte, 8)
		binary.LittleEndian.PutUint64(v.Contents, uint64(iv.(uint)))
	case uint8:
		v.Type = UInt
		v.Contents = make([]byte, 8)
		binary.LittleEndian.PutUint64(v.Contents, uint64(iv.(uint8)))
	case uint16:
		v.Type = UInt
		v.Contents = make([]byte, 8)
		binary.LittleEndian.PutUint64(v.Contents, uint64(iv.(uint16)))
	case uint32:
		v.Type = UInt
		v.Contents = make([]byte, 8)
		binary.LittleEndian.PutUint64(v.Contents, uint64(iv.(uint32)))
	case uint64:
		v.Type = UInt
		v.Contents = make([]byte, 8)
		binary.LittleEndian.PutUint64(v.Contents, uint64(iv.(uint64)))
	case float32, float64:
		err = errors.New("not yet supported type")
	default:
		err = errors.New("not yet supported type")
	}
	return
}
func (v *Value) Decode() (iv interface{}, err error) {
	switch v.Type {
	case Nil:
		iv = nil
	case String:
		iv = string(v.Contents)
	case SInt:
		iv = int64(binary.LittleEndian.Uint64(v.Contents))
	case UInt:
		iv = binary.LittleEndian.Uint64(v.Contents)
	case Boolean:
		if v.Contents[0] == True {
			iv = true
		} else {
			iv = false
		}
	default:
		iv = nil
	}
	return
}

func ToValue(ottoValue otto.Value) (v *Value, err error) {
	v = new(Value)
	valueInterface, err := ottoValue.Export()
	if err != nil {
		return
	}
	err = v.Encode(valueInterface)
	return v, err
}
