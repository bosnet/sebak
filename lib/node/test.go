//
// Functions and types usable only from unit tests
//
package node

import (
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/common/keypair"
)

// Return a new LocalNode with sensible defaults
func NewTestLocalNode0() *LocalNode {
	return NewTestLocalNode(keypair.Random(), &common.Endpoint{Scheme: "memory", Host: "unittests"})
}

// Ditto
func NewTestLocalNode(kp *keypair.Full, endpoint *common.Endpoint) *LocalNode {
	if ret, err := NewLocalNode(kp, endpoint, MakeAlias(kp.Address())); err != nil {
		panic(err)
	} else {
		return ret
	}
}
