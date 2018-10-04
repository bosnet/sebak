package common

import (
	"encoding/json"
	"math"
)

const (
	ConnectMessage                 = "connect"
	TransactionMessage MessageType = "transaction"
	BallotMessage                  = "ballot"
)

type MessageType string

func (t MessageType) String() string {
	return string(t)
}

type Message interface {
	GetType() string
	GetHash() string
	Serialize() ([]byte, error)
	IsWellFormed([]byte, Config) error
	Equal(Message) bool
	Source() string
	// Validate(storage.LevelDBBackend) error
}

// TODO versioning

type NetworkMessage struct {
	Type MessageType
	Data []byte
}

func (t NetworkMessage) Serialize() ([]byte, error) {
	return json.Marshal(t)
}

func (t NetworkMessage) Head(n int) NetworkMessage {
	s := string(t.Data)
	i := math.Min(
		float64(len(s)),
		float64(n),
	)
	return NetworkMessage{
		Type: t.Type,
		Data: []byte(s[:int(i)]),
	}
}

func (t NetworkMessage) IsEmpty() bool {
	return len(t.Data) < 1
}

func NewNetworkMessage(mt MessageType, data []byte) NetworkMessage {
	return NetworkMessage{
		Type: mt,
		Data: data,
	}
}
