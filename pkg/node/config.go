package node

import (
	"github.com/stellar/go/keypair"
	"net/url"
)

type Config struct {
	ListenAddresses []*url.URL
	Validators      []*url.URL

	KeyPair *keypair.Full

	DataDir string

	ChainDbDir string
}
