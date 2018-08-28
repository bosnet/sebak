package value

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/robertkrimen/otto"
)

type Type byte

const (
	Nil     Type = 0x00
	SInt         = 0x01
	UInt         = 0x02
	String       = 0x03
	Boolean      = 0x04
)

const (
	True  = 0x01
	False = 0x00
)

type Value struct {
	Type  Type
	value interface{}
}

func (v *Value) Equal(t *Value) bool {
	if v.Type != t.Type {
		return false
	}
	if v.value != t.value {
		return false
	}
	return true
}

func (v *Value) EqualNative(i interface{}) bool {
	if v.value != i {
		return false
	}
	return true
}

func (v *Value) String() string {
	return fmt.Sprint(v.value)
}

func (v *Value) Value() interface{} {
	return v.value
}

func ToValue(iv interface{}) (v *Value, err error) {
	v = &Value{}

	switch iv.(type) {
	case otto.Value:
		iv, err = iv.(otto.Value).Export()
	case []byte:
		encoded := iv.([]byte)[1:]
		switch Type(iv.([]byte)[0]) {
		case Nil:
			iv = nil
		case String:
			iv = string(encoded)
		case SInt:
			iv = int64(binary.LittleEndian.Uint64(encoded))
		case UInt:
			iv = binary.LittleEndian.Uint64(encoded)
		case Boolean:
			if encoded[0] == True {
				iv = true
			} else {
				iv = false
			}
		default:
			iv = nil
		}
	default:
		// iv = iv
	}

	v.value = iv

	switch iv.(type) {
	case nil:
		v.Type = Nil
	case string:
		v.Type = String
	case bool:
		v.Type = Boolean
	case int:
		v.Type = SInt
		v.value = int64(v.value.(int))
	case int8:
		v.Type = SInt
		v.value = int64(v.value.(int8))
	case int16:
		v.Type = SInt
		v.value = int64(v.value.(int16))
	case int32:
		v.Type = SInt
		v.value = int64(v.value.(int32))
	case int64:
		v.Type = SInt
		v.value = int64(v.value.(int64))
	case uint:
		v.Type = UInt
		v.value = uint64(v.value.(uint))
	case uint8:
		v.Type = UInt
		v.value = uint64(v.value.(uint8))
	case uint16:
		v.Type = UInt
		v.value = uint64(v.value.(uint16))
	case uint32:
		v.Type = UInt
		v.value = uint64(v.value.(uint32))
	case uint64:
		v.Type = UInt
		v.value = uint64(v.value.(uint64))
	case float32, float64:
		v.Type = Nil
		err = errors.New("not yet supported type")
	default:
		v.Type = Nil
		err = errors.New("not yet supported type")
	}
	return
}

func (v *Value) Serialize() (encoded []byte) {

	switch v.Type {
	case Nil:
		encoded = []byte{}
	case SInt:
		encoded = make([]byte, 8)
		binary.LittleEndian.PutUint64(encoded, uint64(v.value.(int64)))
	case UInt:
		encoded = make([]byte, 8)
		binary.LittleEndian.PutUint64(encoded, uint64(v.value.(uint64)))
	case String:
		encoded = []byte(v.value.(string))
	case Boolean:
		if v.value.(bool) {
			encoded = []byte{True}
		} else {
			encoded = []byte{False}
		}
	}

	encoded = append([]byte{byte(v.Type)}, encoded...)

	return
}

func (v *Value) Deserialize(encoded []byte) {

	if v == nil {
		v = new(Value)
	}
	v.Type = Type(encoded[0])
	encoded = encoded[1:]

	switch v.Type {
	case Nil:
		v.value = nil
	case SInt:
		v.value = int64(binary.LittleEndian.Uint64(encoded))
	case UInt:
		v.value = binary.LittleEndian.Uint64(encoded)
	case String:
		v.value = string(encoded)
	case Boolean:
		if bytes.Compare(encoded, []byte{True}) == 0 {
			v.value = true
		} else {
			v.value = false
		}
	}
	return
}
