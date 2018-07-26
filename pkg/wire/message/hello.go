package message

type HelloMessage struct {
	ProtocolVersion uint32

	PubKey PeerId

	Timestamp uint32

	Network string
}
