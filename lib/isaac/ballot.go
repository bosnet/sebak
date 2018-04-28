package consensus

import (
	"encoding/json"

	"github.com/btcsuite/btcutil/base58"
	"github.com/spikeekips/sebak/lib/error"
	"github.com/spikeekips/sebak/lib/storage"
	"github.com/spikeekips/sebak/lib/util"
	"github.com/stellar/go/keypair"
)

type BallotMessage struct {
	Hash    string      `json:"hash"`
	Message interface{} `json:"message"` // if `BallotStateINIT` must have the original `Message`
}

func (bm BallotMessage) IsWellFormed() (err error) {
	if len(bm.Hash) < 1 {
		err = sebak_error.ErrorInvalidHash
		return
	}
	if bm.Message != nil {
		if err = bm.Message.(util.Message).IsWellFormed(); err != nil {
			return
		}
	}

	return nil
}

func (bm BallotMessage) GetHash() string {
	return bm.Hash
}

func (bm BallotMessage) Serialize() ([]byte, error) {
	return json.Marshal(bm)
}

func (bm BallotMessage) String() string {
	encoded, _ := json.MarshalIndent(bm, "", "  ")
	return string(encoded)
}

type Ballot struct {
	H BallotHeader
	B BallotBody
}

func (b Ballot) Serialize() (encoded []byte, err error) {
	if b.GetState() == BallotStateINIT {
		encoded, err = json.Marshal(b)
		return
	}

	newBallot := b
	newBallot.B.Message.Message = nil
	encoded, err = json.Marshal(newBallot)

	return
}

func (b Ballot) String() string {
	encoded, _ := json.MarshalIndent(b, "", "  ")
	return string(encoded)
}

// NewBallotFromMessage creates `Ballot` from `Message`. It needs to be
// `Ballot.IsWellFormed()` and `Ballot.Validate()`.
func NewBallotFromMessage(nodeKey string, m util.Message) (ballot Ballot, err error) {
	message := BallotMessage{
		Hash:    m.GetHash(),
		Message: m,
	}
	body := BallotBody{
		NodeKey:    nodeKey,
		State:      InitialState,
		VotingHole: VotingNOTYET,
		Message:    message,
	}
	ballot = Ballot{
		H: BallotHeader{
			Hash:      base58.Encode(body.MakeHash()),
			Signature: "",
		},
		B: body,
	}

	return
}

func NewBallotFromJSON(b []byte) (ballot Ballot, err error) {
	if err = json.Unmarshal(b, &ballot); err != nil {
		return
	}

	if err = ballot.IsWellFormed(); err != nil {
		return
	}

	return
}

var BallotWellFormedCheckerFuncs = []util.CheckerFunc{
	checkBallotEmptyNodeKey,
	checkBallotEmptyHashMatch,
	checkBallotVerifySignature,
	checkBallotNoVoting,
	checkBallotHasMessage,
	checkBallotValidState,
}

func (b Ballot) IsWellFormed() (err error) {
	if err = util.Checker(BallotWellFormedCheckerFuncs...)(b); err != nil {
		return
	}

	if err = b.GetMessage().IsWellFormed(); err != nil {
		return
	}

	return
}

func (b Ballot) VerifySignature() (err error) {
	err = keypair.MustParse(b.B.NodeKey).Verify(
		[]byte(b.GetHash()),
		base58.Decode(b.H.Signature),
	)
	if err != nil {
		return sebak_error.ErrorSignatureVerificationFailed
	}

	return
}

func (b Ballot) Validate(st *storage.LevelDBBackend) (err error) {
	return
}

func (b Ballot) GetHash() string {
	return b.H.Hash
}

func (b Ballot) GetMessage() BallotMessage {
	return b.B.Message
}

func (b Ballot) GetState() BallotState {
	return b.B.State
}

func (b *Ballot) SetState(state BallotState) {
	b.B.State = state

	return
}

func (b *Ballot) Sign(kp *keypair.Full) {
	if kp.Address() != b.B.NodeKey {
		b.B.NodeKey = kp.Address()
		b.UpdateHash()
	}

	signature, _ := kp.Sign([]byte(b.GetHash()))

	b.H.Signature = base58.Encode(signature)
	return
}

func (b *Ballot) UpdateHash() {
	b.H.Hash = base58.Encode(b.B.MakeHash())

	return
}

func (b *Ballot) Vote(v VotingHole) {
	b.B.VotingHole = v

	return
}

type BallotHeader struct {
	Hash      string `json:"ballot_hash"`
	Signature string `json:"signature"`
}

type BallotBody struct {
	NodeKey    string      `json:"node_key"` // validator's public address
	State      BallotState `json:state`
	VotingHole VotingHole  `json:"voting_hole"`
	Reason     string      `json:"reason"`

	Message BallotMessage `json:"message"`
}

func (bb BallotBody) MakeHash() []byte {
	return util.MustMakeObjectHash(bb)
}
