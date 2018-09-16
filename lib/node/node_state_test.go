package node

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNodeInitState(t *testing.T) {
	require.Equal(t, NodeInitState, StateNONE)
}

func TestNodeStateString(t *testing.T) {
	require.Equal(t, StateNONE.String(), "NONE")
	require.Equal(t, StateBOOTING.String(), "BOOTING")
	require.Equal(t, StateSYNC.String(), "SYNC")
	require.Equal(t, StateCONSENSUS.String(), "CONSENSUS")
	require.Equal(t, StateTERMINATING.String(), "TERMINATING")
}

func TestNodeStateMarshalJSON(t *testing.T) {
	ret, err := StateNONE.MarshalJSON()
	require.Equal(t, err, nil)
	require.Equal(t, "\"NONE\"", string(ret))

	ret, err = StateBOOTING.MarshalJSON()
	require.Equal(t, err, nil)
	require.Equal(t, "\"BOOTING\"", string(ret))

	ret, err = StateSYNC.MarshalJSON()
	require.Equal(t, err, nil)
	require.Equal(t, "\"SYNC\"", string(ret))

	ret, err = StateCONSENSUS.MarshalJSON()
	require.Equal(t, err, nil)
	require.Equal(t, "\"CONSENSUS\"", string(ret))

	ret, err = StateTERMINATING.MarshalJSON()
	require.Equal(t, err, nil)
	require.Equal(t, "\"TERMINATING\"", string(ret))
}

func TestNodeStateUnmarshalJSON(t *testing.T) {
	ns := StateNONE
	nodeStateByteArray, _ := StateNONE.MarshalJSON()
	ns.UnmarshalJSON(nodeStateByteArray)
	require.Equal(t, StateNONE, ns)

	nodeStateByteArray, _ = StateBOOTING.MarshalJSON()
	ns.UnmarshalJSON(nodeStateByteArray)
	require.Equal(t, StateBOOTING, ns)

	nodeStateByteArray, _ = StateSYNC.MarshalJSON()
	ns.UnmarshalJSON(nodeStateByteArray)
	require.Equal(t, StateSYNC, ns)

	nodeStateByteArray, _ = StateCONSENSUS.MarshalJSON()
	ns.UnmarshalJSON(nodeStateByteArray)
	require.Equal(t, StateCONSENSUS, ns)

	nodeStateByteArray, _ = StateTERMINATING.MarshalJSON()
	ns.UnmarshalJSON(nodeStateByteArray)
	require.Equal(t, StateTERMINATING, ns)
}
