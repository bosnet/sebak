package network

import (
	"encoding/json"
	"time"

	"github.com/btcsuite/btcutil/base58"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/common/keypair"
	"boscoin.io/sebak/lib/errors"
	"boscoin.io/sebak/lib/node"
)

type DiscoveryMessage struct {
	H DiscoveryMessageHeader
	B DiscoveryMessageBody
}

type DiscoveryMessageHeader struct {
	Signature string `json:"signature"`
}

type DiscoveryMessageBody struct {
	Created    string            `json:"created"`
	Address    string            `json:"address"`  // LocalNode.Address()
	Endpoint   *common.Endpoint  `json:"endpoint"` // LocalNode.publishEndpoint()
	Validators []*node.Validator `json:"validators"`
}

func NewDiscoveryMessage(localNode *node.LocalNode, validators ...*node.Validator) (dm DiscoveryMessage, err error) {
	// `PublishEndpoint()` must be not empty
	if localNode.PublishEndpoint() == nil {
		err = errors.EndpointNotFound
		return
	}

	dm = DiscoveryMessage{
		H: DiscoveryMessageHeader{},
		B: DiscoveryMessageBody{
			Endpoint:   localNode.PublishEndpoint(),
			Validators: validators,
		},
	}

	return
}

func DiscoveryMessageFromJSON(b []byte) (dm DiscoveryMessage, err error) {
	err = json.Unmarshal(b, &dm)
	return
}

func (db DiscoveryMessageBody) MakeHashString() string {
	return base58.Encode(common.MustMakeObjectHash(db))
}

func (dm DiscoveryMessage) GetHash() string {
	return dm.B.MakeHashString()
}

func (dm DiscoveryMessage) Equal(common.Message) bool {
	return false
}

func (dm DiscoveryMessage) Source() string {
	return dm.B.Address
}

func (dm DiscoveryMessage) Version() string {
	return common.DiscoveryVersionV1
}

func (dm DiscoveryMessage) GetType() common.MessageType {
	return common.DiscoveryMessage
}

func (dm DiscoveryMessage) IsWellFormed(conf common.Config) error {
	if len(dm.H.Signature) < 1 {
		return errors.InvalidMessage
	}
	if len(dm.B.Created) < 1 {
		return errors.InvalidMessage
	}
	if len(dm.B.Address) < 1 {
		return errors.InvalidMessage
	}
	if len(dm.B.Endpoint.String()) < 1 {
		return errors.InvalidMessage
	}

	// check time
	created, err := common.ParseISO8601(dm.B.Created)
	if err != nil {
		return err
	}
	now := time.Now()
	sub := now.Sub(created)
	if sub < (common.DiscoveryMessageCreatedAllowDuration*-1) || sub > common.DiscoveryMessageCreatedAllowDuration {
		return errors.MessageHasIncorrectTime
	}

	return dm.verifySignature(conf.NetworkID)
}

func (dm DiscoveryMessage) verifySignature(networkID []byte) (err error) {
	var kp keypair.KP
	if kp, err = keypair.Parse(dm.B.Address); err != nil {
		return
	}

	hash := dm.B.MakeHashString()
	err = kp.Verify(
		append(networkID, []byte(hash)...),
		base58.Decode(dm.H.Signature),
	)

	return
}

func (dm *DiscoveryMessage) Sign(kp keypair.KP, networkID []byte) {
	dm.B.Created = common.NowISO8601()
	dm.B.Address = kp.Address()

	hash := dm.B.MakeHashString()
	signature, _ := keypair.MakeSignature(kp, networkID, hash)

	dm.H.Signature = base58.Encode(signature)

	return
}

func (dm DiscoveryMessage) Serialize() ([]byte, error) {
	return json.Marshal(dm)
}

func (dm DiscoveryMessage) String() string {
	encoded, _ := json.MarshalIndent(dm, "", "  ")
	return string(encoded)
}

// FilterUndiscovered returns,
//  * not yet registered validators
//  * registered, but endpoint is changed.
// FilterUndiscovered can get the `node.LocalNode.GetValidators()` directly.
func (dm DiscoveryMessage) FilterUndiscovered(validators map[string]*node.Validator) []*node.Validator {
	var undiscovered []*node.Validator

	if rv, ok := validators[dm.B.Address]; ok {
		if rv.Endpoint() == nil || !rv.Endpoint().Equal(dm.B.Endpoint) {
			v, _ := node.NewValidator(dm.B.Address, dm.B.Endpoint, "")
			undiscovered = append(undiscovered, v)
		}
	}

	for _, v := range dm.B.Validators {
		if rv, ok := validators[v.Address()]; !ok {
			continue
		} else if rv.Endpoint() != nil && rv.Endpoint().Equal(v.Endpoint()) {
			continue
		}
		undiscovered = append(undiscovered, v)
	}

	return undiscovered
}
