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

	localNode, _ := NewLocalNode(kp.Address(), endpoint, "node")

	localNode.AddValidators(validator1, validator2)

	tmpByte, err := localNode.MarshalJSON()
	assert.Equal(t, nil, err)

	jsonStr := `"alias":"%s","endpoint":"https://localhost:%s","state":"%s"`
	assert.Equal(t, true, strings.Contains(string(tmpByte), fmt.Sprintf(jsonStr, "node", "5000", "NONE")))
	assert.Equal(t, true, strings.Contains(string(tmpByte), fmt.Sprintf(jsonStr, "v1", "5001", "NONE")))
	assert.Equal(t, true, strings.Contains(string(tmpByte), fmt.Sprintf(jsonStr, "v2", "5002", "NONE")))
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

func TestNodeUnMarshalJSONWithValidator(t *testing.T) {
	kp, _ := keypair.Random()

	endpoint, err := sebakcommon.NewEndpointFromString(fmt.Sprintf("https://localhost:5000?NodeName=n1"))
	assert.Equal(t, nil, err)

	localNode, _ := NewLocalNode(kp.Address(), endpoint, "node")

	localNode.UnmarshalJSON([]byte(
		`{
			"address":"GCEAJXSDCKHQPIIP32OK22VGEUMXSODLIIQHF3J7C4HQBSM6SWQFW37V",
			"alias":"node",
			"endpoint":"https://localhost:5000",
			"state":"NONE",
			"validators":{
				"GBNCCPG47OYMVT5XB4ZKTLUJDCCGYSOPFBKRGLNICUVPKR3RG7RHXECC":
				{
					"address":"GBNCCPG47OYMVT5XB4ZKTLUJDCCGYSOPFBKRGLNICUVPKR3RG7RHXECC",
					"alias":"v2",
					"endpoint":"https://localhost:5002",
					"state":"NONE"
				},
				"GDKAOAG2HR44MYTL4NXA75QFYICAI626UKSC2AEIP4MRJ5QKQ25LHD6V":
				{
					"address":"GDKAOAG2HR44MYTL4NXA75QFYICAI626UKSC2AEIP4MRJ5QKQ25LHD6V",
					"alias":"v1",
					"endpoint":"https://localhost:5001",
					"state":"NONE"
				}
			}
		}`,
	))
	assert.Equal(t, nil, err)

	validators := localNode.GetValidators()
	validator1 := validators["GBNCCPG47OYMVT5XB4ZKTLUJDCCGYSOPFBKRGLNICUVPKR3RG7RHXECC"]
	assert.Equal(t, "v2", validator1.Alias())
	assert.Equal(t, "https://localhost:5002", validator1.Endpoint().String())
	assert.Equal(t, NodeStateNONE, validator1.State())

	validator2 := validators["GDKAOAG2HR44MYTL4NXA75QFYICAI626UKSC2AEIP4MRJ5QKQ25LHD6V"]
	assert.Equal(t, "v1", validator2.Alias())
	assert.Equal(t, "https://localhost:5001", validator2.Endpoint().String())
	assert.Equal(t, NodeStateNONE, validator2.State())

}
