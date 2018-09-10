//
// Provides test utility to mock a network in unittests
//
package network

import (
	"boscoin.io/sebak/lib/node"

	"github.com/stellar/go/keypair"
)

func CreateNewMemoryNetwork() (*keypair.Full, *MemoryNetwork, *node.LocalNode) {
	mn := NewMemoryNetwork()

	kp, _ := keypair.Random()
	localNode, _ := node.NewLocalNode(kp, mn.Endpoint(), "")

	mn.SetLocalNode(localNode)

	return kp, mn, localNode
}
