package consensus

import "github.com/stellar/go/keypair"

type Node interface {
	Keypair() *keypair.Full
	Alias() string
}
