package network

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/common/keypair"
	"boscoin.io/sebak/lib/errors"
	"boscoin.io/sebak/lib/node"
)

func TestDiscoveryMessage(t *testing.T) {
	var networkID []byte = []byte("show-me")

	kp := keypair.Random()
	endpoint := common.MustParseEndpoint("http://1.2.3.4:5678")
	localNode, _ := node.NewLocalNode(kp, endpoint, "")

	var validators []*node.Validator
	{ // add validators
		for i := 0; i < 3; i++ {
			kpv := keypair.Random()
			endpointv := common.MustParseEndpoint(fmt.Sprintf("http://1.2.3.4:567%d", i))
			v, _ := node.NewValidator(kpv.Address(), endpointv, "")
			validators = append(validators, v)
		}
	}
	localNode.AddValidators(validators...)

	{ // localNode.PublishEndpoint() is empty
		localNode.SetPublishEndpoint(nil)
		_, err := NewDiscoveryMessage(localNode)
		require.Error(t, errors.EndpointNotFound, err)
	}

	{ // localNode.PublishEndpoint() is not empty
		localNode.SetPublishEndpoint(endpoint)

		cm, err := NewDiscoveryMessage(localNode, validators...)
		require.NoError(t, err)

		require.Equal(t, len(validators), len(cm.B.Validators))

		// Before signing, `Created` must be empty
		require.Empty(t, cm.B.Created)
	}

	{ // signing
		cm, _ := NewDiscoveryMessage(localNode)
		cm.Sign(localNode.Keypair(), networkID)

		require.NotEmpty(t, cm.H.Signature)
		require.NotEmpty(t, cm.B.Created)
		require.NotEmpty(t, cm.B.Address)
		require.NotEmpty(t, cm.B.Endpoint)

		for _, v := range cm.B.Validators {
			require.NotEmpty(t, v.Address())
			require.NotEmpty(t, v.Alias())
			require.NotEmpty(t, v.Endpoint())
		}
	}

	{ //  and verification
		cm, _ := NewDiscoveryMessage(localNode)
		cm.Sign(localNode.Keypair(), networkID)
		err := cm.verifySignature(networkID)
		require.NoError(t, err)
	}

	{ // from json; verification
		cm, _ := NewDiscoveryMessage(localNode)
		cm.Sign(localNode.Keypair(), networkID)

		b, err := cm.Serialize()
		require.NoError(t, err)

		cmJson, err := DiscoveryMessageFromJSON(b)
		require.NoError(t, err)

		err = cmJson.verifySignature(networkID)
		require.NoError(t, err)
	}
}

func TestDiscoveryMessageUndiscovered(t *testing.T) {
	var networkID []byte = []byte("show-me")

	kp := keypair.Random()
	endpoint := common.MustParseEndpoint("http://1.2.3.4:5678")
	localNode, _ := node.NewLocalNode(kp, endpoint, "")

	var validators []*node.Validator
	{ // add validators
		for i := 0; i < 4; i++ {
			kpv := keypair.Random()
			endpointv := common.MustParseEndpoint(fmt.Sprintf("http://1.2.3.4:567%d", i))
			v, _ := node.NewValidator(kpv.Address(), endpointv, "")
			validators = append(validators, v)

			empty, _ := node.NewValidator(kpv.Address(), nil, "")
			localNode.AddValidators(empty)
		}
	}
	localNode.SetPublishEndpoint(endpoint)

	cm, _ := NewDiscoveryMessage(localNode, validators...)
	cm.Sign(localNode.Keypair(), networkID)

	{ // unregistered validator must be returned
		undiscovered := cm.FilterUndiscovered(localNode.GetValidators())
		require.Equal(t, len(cm.B.Validators), len(undiscovered))
		require.NotNil(t, undiscovered)
	}

	{ // endpoint not updated validators must not be returned
		var emptyEndpoints int
		for _, v := range localNode.GetValidators() {
			if v.Endpoint() == nil {
				emptyEndpoints++
			}
		}
		undiscovered := cm.FilterUndiscovered(localNode.GetValidators())
		require.Equal(t, emptyEndpoints, len(undiscovered))
		require.NotNil(t, undiscovered)
	}
}
