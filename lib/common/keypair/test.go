// Provides utilities to use in test code
package keypair

import (
	stellar "github.com/stellar/go/keypair"
)

//
// Create a new keypair to be used by test code
//
func Random() *Full {
	if kp, err := stellar.Random(); err != nil {
		panic(err)
	} else {
		return kp
	}
}
