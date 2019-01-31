package node

import (
	"fmt"
	"strings"
	"testing"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/common/keypair"

	"github.com/stretchr/testify/require"
)

func TestNodeStateChange(t *testing.T) {
	kp := keypair.Random()
	endpoint := common.MustParseEndpoint("https://localhost:5000?NodeName=n1")

	node := NewTestLocalNode(kp, endpoint)

	require.Equal(t, StateCONSENSUS, node.State())

	node.SetSync()
	require.Equal(t, StateSYNC, node.State())

	node.SetConsensus()
	require.Equal(t, StateCONSENSUS, node.State())
}

func TestNodeMarshalJSON(t *testing.T) {
	kp := keypair.Random()
	endpoint := common.MustParseEndpoint("https://localhost:5000?NodeName=n1")

	marshalNode := NewTestLocalNode(kp, endpoint)
	tmpByte, err := marshalNode.MarshalJSON()
	require.Equal(t, nil, err)

	// alias and address cannot be compared with string literal because these are random generated.
	jsonStr := `"state":"%s"`
	require.Equal(t, true, strings.Contains(string(tmpByte), fmt.Sprintf(jsonStr, "CONSENSUS")), string(tmpByte))

	marshalNode.SetConsensus()
	tmpByte, err = marshalNode.MarshalJSON()
	require.Equal(t, nil, err)
	require.Equal(t, true, strings.Contains(string(tmpByte), fmt.Sprintf(jsonStr, "CONSENSUS")))

	marshalNode.SetSync()
	tmpByte, err = marshalNode.MarshalJSON()
	require.Equal(t, nil, err)
	require.Equal(t, true, strings.Contains(string(tmpByte), fmt.Sprintf(jsonStr, "SYNC")))
}

func TestNodeMarshalJSONWithValidator(t *testing.T) {
	kp := keypair.Random()
	kp2 := keypair.Random()
	kp3 := keypair.Random()

	endpoint := common.MustParseEndpoint("https://localhost:5000?NodeName=n1")
	endpoint2 := common.MustParseEndpoint("https://localhost:5001?NodeName=n2")
	endpoint3 := common.MustParseEndpoint("https://localhost:5002?NodeName=n3")

	validator1, _ := NewValidator(kp2.Address(), endpoint2, "v1")
	validator2, _ := NewValidator(kp3.Address(), endpoint3, "v2")

	localNode, err := NewLocalNode(kp, endpoint, "node")
	require.NoError(t, err)

	localNode.AddValidators(validator1, validator2)

	tmpByte, err := localNode.MarshalJSON()
	require.NoError(t, err)

	require.Equal(t, true, strings.Contains(string(tmpByte), `"alias":"node"`))
	require.Equal(t, true, strings.Contains(string(tmpByte), `"state":"CONSENSUS"`))
	require.Equal(t, true, strings.Contains(string(tmpByte), `"alias":"v1"`))
	require.Equal(t, true, strings.Contains(string(tmpByte), `"endpoint":"https://localhost:5001"`))
	require.Equal(t, true, strings.Contains(string(tmpByte), `"alias":"v2"`))
	require.Equal(t, true, strings.Contains(string(tmpByte), `"endpoint":"https://localhost:5002"`))
}
