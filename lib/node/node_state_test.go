package sebaknode

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNodeInitState(t *testing.T) {
	assert.Equal(t, NodeInitState, NodeStateNONE)
}

func TestNodeStateString(t *testing.T) {
	assert.Equal(t, NodeStateNONE.String(), "NONE")
	assert.Equal(t, NodeStateBOOTING.String(), "BOOTING")
	assert.Equal(t, NodeStateCATCHUP.String(), "CATCHUP")
	assert.Equal(t, NodeStateCONSENSUS.String(), "CONSENSUS")
	assert.Equal(t, NodeStateTERMINATING.String(), "TERMINATING")
}

func TestNodeStateMarshalJSON(t *testing.T) {
	ret, err := NodeStateNONE.MarshalJSON()
	assert.Equal(t, err, nil)
	assert.Equal(t, "\"NONE\"", string(ret))

	ret, err = NodeStateBOOTING.MarshalJSON()
	assert.Equal(t, err, nil)
	assert.Equal(t, "\"BOOTING\"", string(ret))

	ret, err = NodeStateCATCHUP.MarshalJSON()
	assert.Equal(t, err, nil)
	assert.Equal(t, "\"CATCHUP\"", string(ret))

	ret, err = NodeStateCONSENSUS.MarshalJSON()
	assert.Equal(t, err, nil)
	assert.Equal(t, "\"CONSENSUS\"", string(ret))

	ret, err = NodeStateTERMINATING.MarshalJSON()
	assert.Equal(t, err, nil)
	assert.Equal(t, "\"TERMINATING\"", string(ret))
}

func TestNodeStateUnmarshalJSON(t *testing.T) {
	ns := NodeStateNONE
	nodeStateByteArray, _ := NodeStateNONE.MarshalJSON()
	ns.UnmarshalJSON(nodeStateByteArray)
	assert.Equal(t, NodeStateNONE, ns)

	nodeStateByteArray, _ = NodeStateBOOTING.MarshalJSON()
	ns.UnmarshalJSON(nodeStateByteArray)
	assert.Equal(t, NodeStateBOOTING, ns)

	nodeStateByteArray, _ = NodeStateCATCHUP.MarshalJSON()
	ns.UnmarshalJSON(nodeStateByteArray)
	assert.Equal(t, NodeStateCATCHUP, ns)

	nodeStateByteArray, _ = NodeStateCONSENSUS.MarshalJSON()
	ns.UnmarshalJSON(nodeStateByteArray)
	assert.Equal(t, NodeStateCONSENSUS, ns)

	nodeStateByteArray, _ = NodeStateTERMINATING.MarshalJSON()
	ns.UnmarshalJSON(nodeStateByteArray)
	assert.Equal(t, NodeStateTERMINATING, ns)
}
