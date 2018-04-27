package transport

import "net"

type Transport interface {
	Send(net.TCPAddr, []byte) error
	Receive() ([]byte, error)
}
