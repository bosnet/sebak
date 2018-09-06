package network

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/node"

	"github.com/btcsuite/btcutil/base58"
	"github.com/stellar/go/keypair"
)

type DummyMessage struct {
	T    string
	Hash string
	Data string
}

func NewDummyMessage(data string) DummyMessage {
	d := DummyMessage{T: "dummy-message", Data: data}
	d.UpdateHash()

	return d
}

func (m DummyMessage) IsWellFormed([]byte) error {
	return nil
}

func (m DummyMessage) GetType() string {
	return m.T
}

func (m DummyMessage) Equal(n sebakcommon.Message) bool {
	return m.Hash == n.GetHash()
}

func (m DummyMessage) GetHash() string {
	return m.Hash
}

func (m DummyMessage) Source() string {
	return m.Hash
}

func (m *DummyMessage) UpdateHash() {
	m.Hash = base58.Encode(sebakcommon.MustMakeObjectHash(m.Data))
}

func (m DummyMessage) Serialize() ([]byte, error) {
	return json.Marshal(m)
}

func (m DummyMessage) String() string {
	s, _ := json.MarshalIndent(m, "  ", " ")
	return string(s)
}

func DummyMessageFromString(b []byte) (d DummyMessage, err error) {
	if err = json.Unmarshal(b, &d); err != nil {
		return
	}
	return
}

func TestMemoryNetworkCreate(t *testing.T) {
	defer CleanUpMemoryNetwork()

	mh := NewMemoryNetwork()
	stored := getMemoryNetwork(mh.Endpoint())

	if fmt.Sprintf("%p", mh) != fmt.Sprintf("%p", stored) {
		t.Error("failed to remember the memory network")
		return
	}
}

func createNewMemoryNetwork() (*keypair.Full, *MemoryNetwork, *node.LocalNode) {
	mn := NewMemoryNetwork()

	kp, _ := keypair.Random()
	localNode, _ := node.NewLocalNode(kp, mn.Endpoint(), "")

	mn.SetLocalNode(localNode)

	return kp, mn, localNode
}

func TestMemoryNetworkGetClient(t *testing.T) {
	defer CleanUpMemoryNetwork()

	_, s0, _ := createNewMemoryNetwork()

	gotMessage := make(chan Message)
	go func() {
		for message := range s0.ReceiveMessage() {
			gotMessage <- message
		}
	}()

	go s0.Start()

	c0 := s0.GetClient(s0.Endpoint())

	message := NewDummyMessage("findme")
	c0.SendMessage(message)

	select {
	case receivedMessage := <-gotMessage:
		receivedDummy, _ := DummyMessageFromString(receivedMessage.Data)
		if receivedMessage.Type != TransactionMessage {
			t.Error("wrong message type")
		}
		if !message.Equal(receivedDummy) {
			t.Error("got invalid message")
		}
	case <-time.After(1 * time.Second):
		t.Error("failed to get message")
	}
}

func TestMemoryNetworkGetNodeInfo(t *testing.T) {
	defer CleanUpMemoryNetwork()

	_, s0, localNode := createNewMemoryNetwork()

	c0 := s0.GetClient(s0.Endpoint())
	b, err := c0.GetNodeInfo()
	if err != nil {
		t.Error(err)
		return
	}
	v, err := node.NewValidatorFromString(b)
	if err != nil {
		t.Error(err)
		return
	}
	if localNode.Endpoint().String() != v.Endpoint().String() {
		t.Errorf("received node info mismatch; '%s' != '%s'", localNode.Endpoint().String(), v.Endpoint().String())
	}
	if localNode.Address() != v.Address() {
		t.Errorf("received node info mismatch; '%s' != '%s'", localNode.Address(), v.Address())
		return
	}
}

func TestMemoryNetworkConnect(t *testing.T) {
	defer CleanUpMemoryNetwork()

	_, s0, localNode := createNewMemoryNetwork()

	c0 := s0.GetClient(s0.Endpoint())
	b, err := c0.Connect(localNode)
	if err != nil {
		t.Error(err)
		return
	}
	v, err := node.NewValidatorFromString(b)
	if err != nil {
		t.Error(err)
		return
	}
	if localNode.Endpoint().String() != v.Endpoint().String() {
		t.Errorf("received node info mismatch; '%s' != '%s'", localNode.Endpoint().String(), v.Endpoint().String())
	}
	if localNode.Address() != v.Address() {
		t.Errorf("received node info mismatch; '%s' != '%s'", localNode.Address(), v.Address())
		return
	}
}
