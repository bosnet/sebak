package node

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNodeStateString(t *testing.T) {
	require.Equal(t, StateBOOTING.String(), "BOOTING")
	require.Equal(t, StateCONSENSUS.String(), "CONSENSUS")
	require.Equal(t, StateSYNC.String(), "SYNC")
}

func TestNodeStateMarshalJSON(t *testing.T) {
	ret, err := StateBOOTING.MarshalJSON()
	require.Equal(t, err, nil)
	require.Equal(t, "\"BOOTING\"", string(ret))

	ret, err = StateCONSENSUS.MarshalJSON()
	require.Equal(t, err, nil)
	require.Equal(t, "\"CONSENSUS\"", string(ret))

	ret, err = StateSYNC.MarshalJSON()
	require.Equal(t, err, nil)
	require.Equal(t, "\"SYNC\"", string(ret))

}

func TestNodeStateUnmarshalJSON(t *testing.T) {
	ns := StateBOOTING

	nodeStateByteArray, _ := StateBOOTING.MarshalJSON()
	ns.UnmarshalJSON(nodeStateByteArray)
	require.Equal(t, StateBOOTING, ns)

	nodeStateByteArray, _ = StateCONSENSUS.MarshalJSON()
	ns.UnmarshalJSON(nodeStateByteArray)
	require.Equal(t, StateCONSENSUS, ns)

	nodeStateByteArray, _ = StateSYNC.MarshalJSON()
	ns.UnmarshalJSON(nodeStateByteArray)
	require.Equal(t, StateSYNC, ns)
}
