package value

import (
	"github.com/magiconair/properties/assert"
	"github.com/robertkrimen/otto"
	"testing"
)

const (
	MaxUint = ^uint(0)
	MinUint = 0
	MaxInt  = int(MaxUint >> 1)
	MinInt  = -MaxInt - 1

	MaxUint8 = ^uint8(0)
	MinUint8 = 0
	MaxInt8  = int8(MaxUint8 >> 1)
	MinInt8  = -MaxInt8 - 1

	MaxUint16 = ^uint16(0)
	MinUint16 = 0
	MaxInt16  = int16(MaxUint16 >> 1)
	MinInt16  = -MaxInt16 - 1

	MaxUint32 = ^uint32(0)
	MinUint32 = 0
	MaxInt32  = int32(MaxUint32 >> 1)
	MinInt32  = -MaxInt32 - 1

	MaxUint64 = ^uint64(0)
	MinUint64 = 0
	MaxInt64  = int64(MaxUint64 >> 1)
	MinInt64  = -MaxInt64 - 1
)

func TestValue(t *testing.T) {

	testValues := []interface{}{
		true,
		false,

		"BOSCOIN",

		MaxUint,
		MinUint,
		MaxInt,
		MinInt,

		MaxUint8,
		MinUint8,
		MaxInt8,
		MinInt8,

		MaxUint16,
		MinUint16,
		MaxInt16,
		MinInt16,

		MaxUint32,
		MinUint32,
		MaxInt32,
		MinInt32,

		MaxUint64,
		MinUint64,
		MaxInt64,
		MinInt64,
	}

	for _, testValue := range testValues {
		v1, err := ToValue(testValue)
		if err != nil {
			panic(err)
		}
		encoded, err := v1.Serialize()
		if err != nil {
			panic(err)
		}
		v2, err := ToValue(encoded)
		if err != nil {
			panic(err)
		}
		assert.Equal(t, v1, v2)
	}

	var ottoValues []interface{}
	for _, native := range testValues {
		ottoValue, _ := otto.ToValue(native)
		ottoValues = append(ottoValues, ottoValue)
	}

	for _, testValue := range testValues {
		v1, err := ToValue(testValue)
		if err != nil {
			panic(err)
		}
		encoded, err := v1.Serialize()
		if err != nil {
			panic(err)
		}
		v2, err := ToValue(encoded)
		if err != nil {
			panic(err)
		}
		assert.Equal(t, v1, v2)
	}
	for _, testValue := range testValues {
		v1, err := ToValue(testValue)
		if err != nil {
			panic(err)
		}
		v2, err := ToValue(testValue)
		if err != nil {
			panic(err)
		}

		if v1.Equal(v2) != true {
			t.Error("not equal")
		}
	}
	{
		v1, err := ToValue(testValues[0])
		if err != nil {
			panic(err)
		}
		v2, err := ToValue(testValues[1])
		if err != nil {
			panic(err)
		}

		if v1.Equal(v2) == true {
			t.Error("equal")
		}
	}
}
