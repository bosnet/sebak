package node

import (
	"fmt"
	"strings"
	"testing"

	"boscoin.io/sebak/lib/common"

	"github.com/stellar/go/keypair"
	"github.com/stretchr/testify/require"
)

func TestNodeStateChange(t *testing.T) {
	kp, _ := keypair.Random()
	endpoint, err := common.NewEndpointFromString(fmt.Sprintf("https://localhost:5000?NodeName=n1"))
	require.Equal(t, nil, err)

	node, _ := NewLocalNode(kp, endpoint, "")

	require.Equal(t, StateCONSENSUS, node.State())

	node.SetSync()
	require.Equal(t, StateSYNC, node.State())

	node.SetConsensus()
	require.Equal(t, StateCONSENSUS, node.State())
}

func TestNodeMarshalJSON(t *testing.T) {
	kp, _ := keypair.Random()
	endpoint, err := common.NewEndpointFromString(fmt.Sprintf("https://localhost:5000?NodeName=n1"))
	require.Equal(t, nil, err)

	marshalNode, _ := NewLocalNode(kp, endpoint, "")
	tmpByte, err := marshalNode.MarshalJSON()
	require.Equal(t, nil, err)

	// alias and address cannot be compared with string literal because these are random generated.
	jsonStr := `"endpoint":"https://localhost:5000","state":"%s"`
	require.Equal(t, true, strings.Contains(string(tmpByte), fmt.Sprintf(jsonStr, "CONSENSUS")))

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
	kp, _ := keypair.Random()

	endpoint, err := common.NewEndpointFromString(fmt.Sprintf("https://localhost:5000?NodeName=n1"))
	require.Equal(t, nil, err)

	endpoint2, err := common.NewEndpointFromString(fmt.Sprintf("https://localhost:5001?NodeName=n2"))
	require.Equal(t, nil, err)

	endpoint3, err := common.NewEndpointFromString(fmt.Sprintf("https://localhost:5002?NodeName=n3"))
	require.Equal(t, nil, err)

	kp2, _ := keypair.Random()
	kp3, _ := keypair.Random()

	validator1, _ := NewValidator(kp2.Address(), endpoint2, "v1")
	validator2, _ := NewValidator(kp3.Address(), endpoint3, "v2")

	localNode, _ := NewLocalNode(kp, endpoint, "node")

	localNode.AddValidators(validator1, validator2)

	tmpByte, err := localNode.MarshalJSON()
	require.Equal(t, nil, err)

	jsonNodeStr := `"alias":"%s","endpoint":"https://localhost:%s","state":"%s"`
	jsonValidatorStr := `"alias":"%s","endpoint":"https://localhost:%s"`
	require.Equal(t, true, strings.Contains(string(tmpByte), fmt.Sprintf(jsonNodeStr, "node", "5000", "CONSENSUS")))
	require.Equal(t, true, strings.Contains(string(tmpByte), fmt.Sprintf(jsonValidatorStr, "v1", "5001")))
	require.Equal(t, true, strings.Contains(string(tmpByte), fmt.Sprintf(jsonValidatorStr, "v2", "5002")))
}
