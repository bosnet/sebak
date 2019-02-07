//
// Provides test utility to mock a network in unittests
//
package network

import (
	"boscoin.io/sebak/lib/common/keypair"
	"boscoin.io/sebak/lib/node"
)

//
// Create a MemoryNetwork for unittests purpose
//
// If the argument `prev` is `nil`, a whole new network, disconnected from any other network,
// is created. If `prev` is not `nil`, the returned `MemoryNetwork` will be reachable from
// every node `prev` can reference
//
func CreateMemoryNetwork(prev *MemoryNetwork) (*MemoryNetwork, *node.LocalNode) {
	mn := prev.NewMemoryNetwork()
	kp := keypair.Random()
	localNode := node.NewTestLocalNode(kp, mn.Endpoint())
	mn.SetLocalNode(localNode)
	return mn, localNode
}
