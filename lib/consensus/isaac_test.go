package consensus

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetSyncInfoNormal(t *testing.T) {
	vt, err := NewDefaultVotingThresholdPolicy(67)
	require.Nil(t, err)

	is := ISAAC{policy: vt, nodesHeight: make(map[string]uint64)}

	valids := []string{
		"nodeA",
		"nodeB",
		"nodeC",
		"nodeE",
	}

	invalids := []string{
		"nodeD",
	}

	vt.validators = len(valids) + len(invalids)

	is.nodesHeight[valids[0]] = 10
	is.nodesHeight[valids[1]] = 10
	is.nodesHeight[valids[2]] = 10
	is.nodesHeight[invalids[0]] = 3
	is.nodesHeight[valids[3]] = 10

	height, nodeAddrs := is.getSyncInfo()
	require.Equal(t, uint64(10), height)
	require.True(t, contains(nodeAddrs, valids[0]))
	require.True(t, contains(nodeAddrs, valids[1]))
	require.True(t, contains(nodeAddrs, valids[2]))
	require.True(t, contains(nodeAddrs, valids[3]))
}

func TestGetSyncInfoNotFoundValidHeight(t *testing.T) {
	vt, err := NewDefaultVotingThresholdPolicy(67)
	require.Nil(t, err)

	is := ISAAC{policy: vt, nodesHeight: make(map[string]uint64)}

	nodes := []string{
		"nodeA",
		"nodeB",
		"nodeC",
		"nodeE",
	}

	vt.validators = len(nodes)

	is.nodesHeight[nodes[0]] = 10
	is.nodesHeight[nodes[1]] = 20
	is.nodesHeight[nodes[2]] = 30
	is.nodesHeight[nodes[3]] = 40

	height, nodeAddrs := is.getSyncInfo()
	require.Equal(t, uint64(1), height)
	require.Empty(t, nodeAddrs)
}

func TestGetSyncInfoGenesis(t *testing.T) {
	vt, err := NewDefaultVotingThresholdPolicy(67)
	require.Nil(t, err)

	is := ISAAC{policy: vt, nodesHeight: make(map[string]uint64)}

	nodes := []string{
		"nodeA",
		"nodeB",
		"nodeC",
		"nodeE",
	}

	vt.validators = len(nodes)

	is.nodesHeight[nodes[0]] = 1
	is.nodesHeight[nodes[1]] = 1
	is.nodesHeight[nodes[2]] = 1
	is.nodesHeight[nodes[3]] = 1

	height, nodeAddrs := is.getSyncInfo()
	require.Equal(t, uint64(1), height)
	require.Empty(t, nodeAddrs)
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
