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

	node, _ := NewLocalNode(kp.Address(), endpoint, "")

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

	marshalNode, _ := NewLocalNode(kp.Address(), endpoint, "")
	tmpByte, err := marshalNode.MarshalJSON()
	assert.Equal(t, nil, err)

	// alias and address cannot be compared with string literal because these are random generated.
	jsonStr := "\"endpoint\":\"https://localhost:5000\",\"state\":\"%s\""
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

func TestNodeUnmarshalJSON(t *testing.T) {
	kp, _ := keypair.Random()
	endpoint, err := sebakcommon.NewEndpointFromString(fmt.Sprintf("https://localhost:5000?NodeName=n1"))
	assert.Equal(t, nil, err)

	unmarshalNode, _ := NewLocalNode(kp.Address(), endpoint, "")
	jsonStr := "{\"address\":\"GCSDFPEBQJ7FWAPTZDXNVDYPDDBSAPFZU5MKKM6I35JXJPYD3TFRST7F\",\"alias\":\"GCSD.3TFR\",\"endpoint\":\"https://localhost:5000\",\"state\":\"%s\"}"
	unmarshalNode.UnmarshalJSON([]byte(fmt.Sprintf(jsonStr, "NONE")))

	assert.Equal(t, "GCSDFPEBQJ7FWAPTZDXNVDYPDDBSAPFZU5MKKM6I35JXJPYD3TFRST7F", unmarshalNode.Address())
	assert.Equal(t, "GCSD.3TFR", unmarshalNode.Alias())
	assert.Equal(t, "https://localhost:5000", unmarshalNode.Endpoint().String())
	assert.Equal(t, NodeStateNONE, unmarshalNode.State())

	unmarshalNode.UnmarshalJSON([]byte(fmt.Sprintf(jsonStr, "BOOTING")))
	assert.Equal(t, NodeStateBOOTING, unmarshalNode.State())

	unmarshalNode.UnmarshalJSON([]byte(fmt.Sprintf(jsonStr, "CATCHUP")))
	assert.Equal(t, NodeStateCATCHUP, unmarshalNode.State())

	unmarshalNode.UnmarshalJSON([]byte(fmt.Sprintf(jsonStr, "CONSENSUS")))
	assert.Equal(t, NodeStateCONSENSUS, unmarshalNode.State())

	unmarshalNode.UnmarshalJSON([]byte(fmt.Sprintf(jsonStr, "TERMINATING")))
	assert.Equal(t, NodeStateTERMINATING, unmarshalNode.State())
}
