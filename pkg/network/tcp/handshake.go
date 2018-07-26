package tcp

import (
	"boscoin.io/sebak/pkg/wire/message"
	"net"
	"time"
)

const (
	connectTimeout = time.Second * 5
)

func (o *NetworkManager) handshake(conn net.Conn, inbound bool) (*peer, error) {
	var err error
	var their *message.HelloMessage
	our := &message.HelloMessage{
		ProtocolVersion: message.ProtocolVersion,
		PubKey:          o.params.PubKey,
		Timestamp:       uint32(time.Now().Second()),
		Network:         o.params.NetworkPassphrase,
	}

	if inbound {
		if their, err = o.readHelloMessage(conn); err != nil {
			return nil, err
		} else if err := o.writeHelloMessage(conn, our); err != nil {
			return nil, err
		}
	} else {
		if err = o.writeHelloMessage(conn, our); err != nil {
			return nil, err
		} else if their, err = o.readHelloMessage(conn); err != nil {
			return nil, err
		}
	}

	if our.PubKey == their.PubKey {
		conn.Close()
		return nil, nil
	}

	return newPeer(inbound, conn, their), nil
}

func (o *NetworkManager) writeHelloMessage(conn net.Conn, msg *message.HelloMessage) error {
	_, err := o.protocol.Pack(conn, msg)
	return err
}

func (o *NetworkManager) readHelloMessage(conn net.Conn) (*message.HelloMessage, error) {
	if their, err := o.protocol.Unpack(conn); err != nil {
		return nil, err
	} else {
		return their.(*message.HelloMessage), nil
	}
}
