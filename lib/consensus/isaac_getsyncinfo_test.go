package consensus

import (
	"testing"

	logging "github.com/inconshreveable/log15"
	"github.com/stretchr/testify/require"
)

func TestGetSyncInfoNormal(t *testing.T) {
	vt, err := NewDefaultVotingThresholdPolicy(67)
	require.NoError(t, err)

	is := ISAAC{
		policy:      vt,
		nodesHeight: make(map[string]uint64),
		log:         logging.New("module", "consensus"),
	}

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

	height, nodeAddrs, err := is.GetSyncInfo()
	require.NoError(t, err)
	require.Equal(t, uint64(10), height)
	require.True(t, contains(nodeAddrs, valids[0]))
	require.True(t, contains(nodeAddrs, valids[1]))
	require.True(t, contains(nodeAddrs, valids[2]))
	require.True(t, contains(nodeAddrs, valids[3]))
}
func TestGetSyncInfoBeforeFull(t *testing.T) {
	vt, err := NewDefaultVotingThresholdPolicy(67)
	require.NoError(t, err)

	is := ISAAC{
		policy:      vt,
		nodesHeight: make(map[string]uint64),
		log:         logging.New("module", "consensus"),
	}

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
	is.nodesHeight[valids[3]] = 10

	height, nodeAddrs, err := is.GetSyncInfo()
	require.NoError(t, err)
	require.Equal(t, uint64(10), height)
	require.True(t, contains(nodeAddrs, valids[0]))
	require.True(t, contains(nodeAddrs, valids[1]))
	require.True(t, contains(nodeAddrs, valids[2]))
	require.True(t, contains(nodeAddrs, valids[3]))
}

func TestGetSyncInfoFoundSmallestHeight(t *testing.T) {
	vt, err := NewDefaultVotingThresholdPolicy(67)
	require.NoError(t, err)

	is := ISAAC{
		policy:      vt,
		nodesHeight: make(map[string]uint64),
		log:         logging.New("module", "consensus"),
	}

	nodes := []string{
		"nodeA",
		"nodeB",
		"nodeC",
		"nodeD",
		"nodeE",
	}

	vt.validators = len(nodes)

	is.nodesHeight[nodes[0]] = 32
	is.nodesHeight[nodes[1]] = 33
	is.nodesHeight[nodes[2]] = 33
	is.nodesHeight[nodes[3]] = 34
	is.nodesHeight[nodes[4]] = 35

	is.log = logging.New("module", "consensus")
	height, nodeAddrs, err := is.GetSyncInfo()
	require.NoError(t, err)
	require.Equal(t, uint64(33), height)

	require.True(t, contains(nodeAddrs, nodes[1]))
	require.True(t, contains(nodeAddrs, nodes[2]))
	require.True(t, contains(nodeAddrs, nodes[3]))
	require.True(t, contains(nodeAddrs, nodes[4]))
}

func TestGetSyncInfoGenesis(t *testing.T) {
	vt, err := NewDefaultVotingThresholdPolicy(67)
	require.NoError(t, err)

	is := ISAAC{
		policy:      vt,
		nodesHeight: make(map[string]uint64),
		log:         logging.New("module", "consensus"),
	}

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

	is.log = logging.New("module", "consensus")
	height, nodeAddrs, err := is.GetSyncInfo()
	require.NoError(t, err)
	require.Equal(t, uint64(1), height)
	require.Equal(t, 3, len(nodeAddrs))
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
