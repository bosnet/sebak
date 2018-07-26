package message

import (
	"boscoin.io/sebak/pkg/wire"
)

const (
	ProtocolVersion = 1
)

func newMsgId(groupId byte, seq byte) wire.MsgId {
	return [4]byte{groupId, seq, 0, 0}
}

func RegisterAllMessages(proto wire.Protocol) wire.Protocol {
	proto.Register(newMsgId(0, 0), &HelloMessage{})
	proto.Register(newMsgId(4, 1), &BlockRequestMessage{})
	proto.Register(newMsgId(4, 2), &BlockResponseMessage{})
	proto.Register(newMsgId(4, 3), &BlockHeightRequestMessage{})
	proto.Register(newMsgId(4, 4), &BlockHeightResponseMessage{})

	return proto
}
