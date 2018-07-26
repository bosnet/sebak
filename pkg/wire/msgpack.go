package wire

import (
	"bytes"
	"encoding/binary"
	"github.com/pkg/errors"
	"gopkg.in/vmihailenco/msgpack.v3"
	"io"
	"reflect"
	"sync"
)

type MsgPackProtocol struct {
	lock     sync.Mutex
	sequence uint32

	messages   map[MsgId]*messageInfo
	messageIds map[string]MsgId
}

type msgpackPacketHeader struct {
	MsgId MsgId

	Sequence uint32
}

type messageInfo struct {
	msgId MsgId

	msgType reflect.Type
}

func NewMsgpackProtocol() *MsgPackProtocol {
	return &MsgPackProtocol{
		messages:   make(map[MsgId]*messageInfo),
		messageIds: make(map[string]MsgId),
	}
}

func (o *MsgPackProtocol) Register(msgId MsgId, m interface{}) {
	if m == nil {
		panic(errors.New("nil is not supported"))
	}

	o.lock.Lock()
	defer o.lock.Unlock()

	if _, ok := o.messages[msgId]; ok {
		panic(errors.Errorf("conflict msgId code : %s", msgId))
	}

	t := reflect.TypeOf(m)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		panic(errors.Errorf("%s type is not supported", t.Kind()))
	}

	o.messageIds[t.Name()] = msgId
	o.messages[msgId] = &messageInfo{
		msgId:   msgId,
		msgType: t,
	}
}

func (o *MsgPackProtocol) lookup(m interface{}) *messageInfo {
	if m != nil {
		t := reflect.TypeOf(m)
		if t.Kind() == reflect.Ptr {
			t = t.Elem()
		}
		if m, ok := o.messageIds[t.Name()]; ok {
			return o.messages[m]
		}
	}

	return nil
}

func (o *MsgPackProtocol) Pack(w io.Writer, msg interface{}) (int, error) {
	info := o.lookup(msg)

	if info == nil {
		panic(errors.Errorf("%s type is not registered", reflect.TypeOf(msg)))
	}

	var b bytes.Buffer
	encoder := msgpack.NewEncoder(&b)
	encoder.StructAsArray(true)
	encoder.Encode(&msgpackPacketHeader{
		MsgId:    info.msgId,
		Sequence: o.nextSequence(),
	})
	encoder.Encode(msg)

	numBytes := 0
	if n, err := w.Write(o.packHeader(b.Len())); err != nil {
		return -1, err
	} else {
		numBytes += n
	}

	if n, err := w.Write(b.Bytes()); err != nil {
		return -1, err
	} else {
		numBytes += n
	}

	return numBytes, nil
}

func (o *MsgPackProtocol) packHeader(numBytes int) []byte {
	// TODO: compression, signature validation, etc.
	// 0 : magic code
	// 1-3 : flags (ex: compression, encrypt, payload type, etc.)
	// 4-7 : packet bytes length
	var b bytes.Buffer
	binary.Write(&b, binary.BigEndian, byte('B'))
	binary.Write(&b, binary.BigEndian, [3]byte{0, 0, 0})
	binary.Write(&b, binary.BigEndian, uint32(numBytes))

	return b.Bytes()
}

func (o *MsgPackProtocol) Unpack(r io.Reader) (interface{}, error) {
	_, numBytes := o.unpackHeader(r)

	if numBytes >= 10*1024*1024 {
		return nil, errors.New("payload is too large")
	}

	payload := make([]byte, numBytes)

	if _, err := io.ReadFull(r, payload); err != nil {
		return nil, err
	}

	header := &msgpackPacketHeader{}
	if err := msgpack.Unmarshal(payload, header); err != nil {
		return nil, err
	}

	v := reflect.New(o.messages[header.MsgId].msgType).Interface()
	if err := msgpack.Unmarshal(payload[8:], &v); err != nil {
		return nil, err
	}

	return v, nil
}

func (o *MsgPackProtocol) unpackHeader(r io.Reader) (uint64, uint32) {
	var header [8]byte
	io.ReadFull(r, header[:])
	return 0, binary.BigEndian.Uint32(header[4:])
}

func (o *MsgPackProtocol) nextSequence() uint32 {
	o.lock.Lock()
	defer o.lock.Unlock()

	o.sequence += 1

	return o.sequence
}
