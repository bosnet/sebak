//
// Struct that bridges together components of a node
//
// NodeRunner bridges together the connection, storage and `LocalNode`.
// In this regard, it can be seen as a single node, and is used as such
// in unit tests.
//
package sebak

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestConfigurationDefault(t *testing.T) {
	n := NewNodeRunnerConfiguration()
	assert.Equal(t, n.TimeoutINIT, 2*time.Second)
	assert.Equal(t, n.TimeoutSIGN, 2*time.Second)
	assert.Equal(t, n.TimeoutACCEPT, 2*time.Second)
	assert.Equal(t, n.TimeoutALLCONFIRM, 2*time.Second)
	assert.Equal(t, n.TransactionsLimit, 1000)
}

func TestConfigurationSetAndGet(t *testing.T) {
	n := NewNodeRunnerConfiguration()
	n.SetINIT(3).SetSIGN(1).SetACCEPT(1).SetALLCONFIRM(2).SetTxLimit(500)

	assert.Equal(t, n.TimeoutINIT, 3*time.Second)
	assert.Equal(t, n.TimeoutSIGN, 1*time.Second)
	assert.Equal(t, n.TimeoutACCEPT, 1*time.Second)
	assert.Equal(t, n.TimeoutALLCONFIRM, 2*time.Second)
	assert.Equal(t, n.TransactionsLimit, 500)
}
