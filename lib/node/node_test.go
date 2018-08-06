package sebaknode

import (
	"fmt"
	"strings"
	"testing"

	"boscoin.io/sebak/lib/common"

	"github.com/stellar/go/keypair"
	"github.com/stretchr/testify/assert"
)

func TestNodeStateChange(t *testing.T) {
	kp, _ := keypair.Random()
	endpoint, err := sebakcommon.NewEndpointFromString(fmt.Sprintf("https://localhost:5000?NodeName=n1"))
	assert.Equal(t, nil, err)

	node, _ := NewLocalNode(kp, endpoint, "")

	assert.Equal(t, NodeStateNONE, node.State())

	node.SetBooting()
	assert.Equal(t, NodeStateBOOTING, node.State())

	node.SetCatchup()
	assert.Equal(t, NodeStateCATCHUP, node.State())

	node.SetConsensus()
	assert.Equal(t, NodeStateCONSENSUS, node.State())

	node.SetTerminating()
	assert.Equal(t, NodeStateTERMINATING, node.State())

}

func TestNodeMarshalJSON(t *testing.T) {
	kp, _ := keypair.Random()
	endpoint, err := sebakcommon.NewEndpointFromString(fmt.Sprintf("https://localhost:5000?NodeName=n1"))
	assert.Equal(t, nil, err)

	marshalNode, _ := NewLocalNode(kp, endpoint, "")
	tmpByte, err := marshalNode.MarshalJSON()
	assert.Equal(t, nil, err)

	// alias and address cannot be compared with string literal because these are random generated.
	jsonStr := `"endpoint":"https://localhost:5000","state":"%s"`
	assert.Equal(t, true, strings.Contains(string(tmpByte), fmt.Sprintf(jsonStr, "NONE")))

	marshalNode.SetBooting()
	tmpByte, err = marshalNode.MarshalJSON()
	assert.Equal(t, nil, err)
	assert.Equal(t, true, strings.Contains(string(tmpByte), fmt.Sprintf(jsonStr, "BOOTING")))

	marshalNode.SetCatchup()
	tmpByte, err = marshalNode.MarshalJSON()
	assert.Equal(t, nil, err)
	assert.Equal(t, true, strings.Contains(string(tmpByte), fmt.Sprintf(jsonStr, "CATCHUP")))

	marshalNode.SetConsensus()
	tmpByte, err = marshalNode.MarshalJSON()
	assert.Equal(t, nil, err)
	assert.Equal(t, true, strings.Contains(string(tmpByte), fmt.Sprintf(jsonStr, "CONSENSUS")))

	marshalNode.SetTerminating()
	tmpByte, err = marshalNode.MarshalJSON()
	assert.Equal(t, nil, err)
	assert.Equal(t, true, strings.Contains(string(tmpByte), fmt.Sprintf(jsonStr, "TERMINATING")))
}

func TestNodeMarshalJSONWithValidator(t *testing.T) {
	kp, _ := keypair.Random()

	endpoint, err := sebakcommon.NewEndpointFromString(fmt.Sprintf("https://localhost:5000?NodeName=n1"))
	assert.Equal(t, nil, err)

	endpoint2, err := sebakcommon.NewEndpointFromString(fmt.Sprintf("https://localhost:5001?NodeName=n2"))
	assert.Equal(t, nil, err)

	endpoint3, err := sebakcommon.NewEndpointFromString(fmt.Sprintf("https://localhost:5002?NodeName=n3"))
	assert.Equal(t, nil, err)

	kp2, _ := keypair.Random()
	kp3, _ := keypair.Random()

	validator1, _ := NewValidator(kp2.Address(), endpoint2, "v1")
	validator2, _ := NewValidator(kp3.Address(), endpoint3, "v2")

	localNode, _ := NewLocalNode(kp, endpoint, "node")

	localNode.AddValidators(validator1, validator2)

	tmpByte, err := localNode.MarshalJSON()
	assert.Equal(t, nil, err)

	jsonStr := `"alias":"%s","endpoint":"https://localhost:%s","state":"%s"`
	assert.Equal(t, true, strings.Contains(string(tmpByte), fmt.Sprintf(jsonStr, "node", "5000", "NONE")))
	assert.Equal(t, true, strings.Contains(string(tmpByte), fmt.Sprintf(jsonStr, "v1", "5001", "NONE")))
	assert.Equal(t, true, strings.Contains(string(tmpByte), fmt.Sprintf(jsonStr, "v2", "5002", "NONE")))
}
