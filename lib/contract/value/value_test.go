package value

import (
	"testing"
	"github.com/magiconair/properties/assert"
)
const (
	MaxUint= ^uint(0)
	MinUint= 0
	MaxInt= int(MaxUint >> 1)
	MinInt= -MaxInt - 1

	MaxUint8= ^uint8(0)
	MinUint8= 0
	MaxInt8= int8(MaxUint8 >> 1)
	MinInt8= -MaxInt8 - 1

	MaxUint16= ^uint16(0)
	MinUint16= 0
	MaxInt16= int16(MaxUint16 >> 1)
	MinInt16= -MaxInt16 - 1

	MaxUint32= ^uint32(0)
	MinUint32= 0
	MaxInt32= int32(MaxUint32 >> 1)
	MinInt32= -MaxInt32 - 1

	MaxUint64= ^uint64(0)
	MinUint64= 0
	MaxInt64= int64(MaxUint64 >> 1)
	MinInt64= -MaxInt64 - 1
)

func TestValue(t *testing.T){
	//Boolean
	{
		{
			var v= new(Value)
			var native= true
			v.Encode(native)
			decodedValue, _ := v.Decode()
			assert.Equal(t, decodedValue, native)
		}
		{
			var v= new(Value)
			var native= false
			v.Encode(native)
			decodedValue, _ := v.Decode()
			assert.Equal(t, decodedValue, native)
		}
	}
	//Nil
	{
		{
			var v= new(Value)
			var native interface{} = nil
			v.Encode(native)
			decodedValue, _ := v.Decode()
			assert.Equal(t, decodedValue, native)
		}
	}
	//String
	{
		{
			var v= new(Value)
			var native = "GAZUA BOSCOIN"
			v.Encode(native)
			decodedValue, _ := v.Decode()
			assert.Equal(t, decodedValue, native)
		}
	}

	//Signed Integer
	{
		{
			var v= new(Value)
			var native = MaxInt
			v.Encode(native)
			decodedValue, _ := v.Decode()
			assert.Equal(t, decodedValue, int64(native))
		}
		{
			var v= new(Value)
			var native = MaxInt8
			v.Encode(native)
			decodedValue, _ := v.Decode()
			assert.Equal(t, decodedValue, int64(native))
		}
		{
			var v= new(Value)
			var native = MaxInt16
			v.Encode(native)
			decodedValue, _ := v.Decode()
			assert.Equal(t, decodedValue, int64(native))
		}
		{
			var v= new(Value)
			var native = MaxInt32
			v.Encode(native)
			decodedValue, _ := v.Decode()
			assert.Equal(t, decodedValue, int64(native))
		}
		{
			var v= new(Value)
			var native = MaxInt64
			v.Encode(native)
			decodedValue, _ := v.Decode()
			assert.Equal(t, decodedValue, int64(native))
		}
	}

	//Unsigned Integer
	{
		{
			var v= new(Value)
			var native = MaxUint
			v.Encode(native)
			decodedValue, _ := v.Decode()
			assert.Equal(t, decodedValue, uint64(native))
		}
		{
			var v= new(Value)
			var native = MaxUint8
			v.Encode(native)
			decodedValue, _ := v.Decode()
			assert.Equal(t, decodedValue, uint64(native))
		}
		{
			var v= new(Value)
			var native = MaxUint16
			v.Encode(native)
			decodedValue, _ := v.Decode()
			assert.Equal(t, decodedValue, uint64(native))
		}
		{
			var v= new(Value)
			var native = MaxUint32
			v.Encode(native)
			decodedValue, _ := v.Decode()
			assert.Equal(t, decodedValue, uint64(native))
		}
		{
			var v= new(Value)
			var native = MaxUint64
			v.Encode(native)
			decodedValue, _ := v.Decode()
			assert.Equal(t, decodedValue, uint64(native))
		}
	}
}