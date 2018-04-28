package consensus

import "github.com/stellar/go/keypair"

type Node interface {
	GetKeypair() *keypair.Full
	GetAlias() string
}
