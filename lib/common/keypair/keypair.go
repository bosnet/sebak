//
// Encapsulate Stellar's keypair package
//
// Provides additional wrapper and convenience functions,
// suited for usage within Sebak
//
package keypair

import (
	stellar "github.com/stellar/go/keypair"
)

// Aliases to stellar types
type Full = stellar.Full
type KP = stellar.KP

// Aliases to stellar functions
var Master = stellar.Master
var Parse = stellar.Parse
var RandomCanFail = stellar.Random

// MakeSignature makes signature from given hash string
func MakeSignature(kp KP, networkID []byte, hash string) ([]byte, error) {
	return kp.Sign(append(networkID, []byte(hash)...))
}
