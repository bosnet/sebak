package wire

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"testing"
)

type testMessage struct {
	Text string
	Num  int32
}

type header struct {
	Num int32
}

type body struct {
	Num int32
}

type ant struct {
	Header header
	Body1  body
	Body2  body
	Body3  body
}

func TestProtocol_Serialize(t *testing.T) {
	proto := NewMsgpackProtocol()
	proto.Register([4]byte{0, 0, 0, 0}, &testMessage{})
	proto.Register([4]byte{0, 0, 0, 1}, &ant{})

	var w bytes.Buffer
	proto.Pack(&w, &ant{
		Header: header{1},
		Body1:  body{2},
		Body2:  body{3},
		Body3:  body{4},
	})

	v, err := proto.Unpack(bytes.NewReader(w.Bytes()))

	if err != nil {
		assert.Fail(t, err.Error())
	}

	if msg, ok := v.(*ant); ok {
		assert.Equal(t, int32(1), msg.Header.Num)
		assert.Equal(t, int32(2), msg.Body1.Num)
		assert.Equal(t, int32(3), msg.Body2.Num)
		assert.Equal(t, int32(4), msg.Body3.Num)
	} else {
		assert.Fail(t, "msg is invalid")
	}
}
