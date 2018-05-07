package sebak

type Transport interface {
	Start() error
	Ready() error
	Send(Node, []byte) error
	SendRaw(string, []byte) error
	Receive() ([]byte, error)

	Endpoint() string
}
