package sebak

import (
	"encoding/json"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/error"
	"boscoin.io/sebak/lib/storage"
	"github.com/btcsuite/btcutil/base58"
	"github.com/stellar/go/keypair"
)

type BallotData struct {
	//Hash    string      `json:"hash"`
	Data interface{} `json:"data"` // if `BallotStateINIT` must have the original `Message`
}

func (bd BallotData) IsEmpty() bool {
	return bd.Data == nil
}

func (bd BallotData) Message() sebakcommon.Message {
	if bd.IsEmpty() {
		return nil
	}

	return bd.Data.(sebakcommon.Message)
}

func (bd BallotData) Serialize() ([]byte, error) {
	if bd.Data == nil {
		return []byte{}, nil
	}
	return json.Marshal(bd.Data)
}

func (bd BallotData) String() string {
	if bd.Data == nil {
		return ""
	}
	encoded, _ := json.MarshalIndent(bd, "", "  ")
	return string(encoded)
}

// TODO versioning

type Ballot struct {
	T string
	H BallotHeader
	B BallotBody
	D BallotData
}

func (b Ballot) Clone() Ballot {
	body := BallotBody{
		Hash:       b.B.Hash,
		NodeKey:    b.B.NodeKey,
		State:      b.B.State,
		VotingHole: b.B.VotingHole,
	}
	return Ballot{
		T: b.T,
		H: BallotHeader{
			Hash:      b.H.Hash,
			Signature: b.H.Signature,
		},
		B: body,
		D: b.D,
	}
}

// NOTE(Ballot.Serialize): `Ballot.Serialize`: the original idea was this, every
// time to transfer the ballot with tx message is waste of network, so the tx
// message will be received at the first time(at BallotStateINIT), but if node
// get consensus after BallotStateINIT, the node has no way to find the original
// tx message.
// TODO(Ballot.Serialize): `Ballot.Serialize`: find the way to reduce the ballot
// size without message.

func (b Ballot) IsEmpty() bool {
	return len(b.GetType()) < 1
}

func (b Ballot) Serialize() (encoded []byte, err error) {
	//if b.State() == sebakcommon.BallotStateINIT {
	//	encoded, err = json.Marshal(b)
	//	return
	//}

	//newBallot := b
	//newBallot.D.Data = nil
	encoded, err = json.Marshal(b)

	return
}

func (b Ballot) String() string {
	encoded, _ := json.MarshalIndent(b, "", "  ")
	return string(encoded)
}

func (b Ballot) CanFitInVotingBox() (ret bool) {
	switch b.State() {
	case sebakcommon.BallotStateSIGN:
	case sebakcommon.BallotStateACCEPT:
		ret = true
	default:
		ret = false
	}

	return
}

func (b Ballot) CanFitInWaitingBox() bool {
	return b.State() == sebakcommon.BallotStateINIT
}

// NewBallotFromMessage creates `Ballot` from `Message`. It needs to be
// `Ballot.IsWellFormed()` and `Ballot.Validate()`.
func NewBallotFromMessage(nodeKey string, m sebakcommon.Message) (ballot Ballot, err error) {
	body := BallotBody{
		Hash:       m.GetHash(),
		NodeKey:    nodeKey,
		State:      sebakcommon.InitialState,
		VotingHole: VotingNOTYET,
	}
	data := BallotData{
		Data: m,
	}
	ballot = Ballot{
		T: "ballot",
		H: BallotHeader{
			Hash:      base58.Encode(body.MakeHash()),
			Signature: "",
		},
		B: body,
		D: data,
	}

	return
}

func NewBallotFromJSON(b []byte) (ballot Ballot, err error) {
	if err = json.Unmarshal(b, &ballot); err != nil {
		return
	}

	if ballot.Data().IsEmpty() {
		return
	}

	a, _ := ballot.Data().Serialize()

	// TODO BallotMessage should load message by it's `GetType()`
	tx, _ := NewTransactionFromJSON(a)
	ballot.SetData(tx)
	//if err = ballot.IsWellFormed(); err != nil {
	//	return
	//}

	return
}

var BallotWellFormedCheckerFuncs = []sebakcommon.CheckerFunc{
	checkBallotEmptyNodeKey,
	checkBallotEmptyHashMatch,
	checkBallotVerifySignature,
	checkBallotNoVoting,
	checkBallotHasMessage,
	checkBallotValidState,
}

func (b Ballot) IsWellFormed(networkID []byte) (err error) {
	checker := &BallotChecker{
		DefaultChecker: sebakcommon.DefaultChecker{BallotWellFormedCheckerFuncs},
		Ballot:         b,
		NetworkID:      networkID,
	}
	if err = sebakcommon.RunChecker(checker, sebakcommon.DefaultDeferFunc); err != nil {
		return
	}

	return
}

func (b Ballot) VerifySignature(networkID []byte) (err error) {
	err = keypair.MustParse(b.B.NodeKey).Verify(
		append(networkID, []byte(b.GetHash())...),
		base58.Decode(b.H.Signature),
	)
	if err != nil {
		return sebakerror.ErrorSignatureVerificationFailed
	}

	return
}

func (b Ballot) Validate(st *sebakstorage.LevelDBBackend) (err error) {
	return
}

func (b Ballot) GetType() string {
	return b.T
}

func (b Ballot) Equal(m sebakcommon.Message) bool {
	return b.H.Hash == m.GetHash()
}

func (b Ballot) GetHash() string {
	return b.H.Hash
}

func (b Ballot) MessageHash() string {
	return b.B.Hash
}

func (b Ballot) Data() BallotData {
	return b.D
}

func (b *Ballot) SetData(m sebakcommon.Message) {
	b.D.Data = m
}

func (b Ballot) State() sebakcommon.BallotState {
	return b.B.State
}

func (b *Ballot) SetState(state sebakcommon.BallotState) {
	b.B.State = state

	return
}

func (b *Ballot) Sign(kp *keypair.Full, networkID []byte) {
	if kp.Address() != b.B.NodeKey {
		b.B.NodeKey = kp.Address()
	}

	b.UpdateHash()
	signature, _ := kp.Sign(append(networkID, []byte(b.GetHash())...))

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
	Hash       string                  `json:"hash"`
	NodeKey    string                  `json:"node_key"` // validator's public address
	State      sebakcommon.BallotState `json:"state"`
	VotingHole VotingHole              `json:"voting_hole"`
	Reason     string                  `json:"reason"`
}

func (bb BallotBody) MakeHash() []byte {
	return sebakcommon.MustMakeObjectHash(bb)
}

type BallotBoxes struct {
	sebakcommon.SafeLock

	Results map[ /* `Message.GetHash()`*/ string]*VotingResult

	WaitingBox  *BallotBox
	VotingBox   *BallotBox
	ReservedBox *BallotBox

	Messages map[ /* `Message.GetHash()`*/ string]sebakcommon.Message
}

func NewBallotBoxes() *BallotBoxes {
	return &BallotBoxes{
		Results:     map[string]*VotingResult{},
		WaitingBox:  NewBallotBox(),
		VotingBox:   NewBallotBox(),
		ReservedBox: NewBallotBox(),
		Messages:    map[string]sebakcommon.Message{},
	}
}

func (b *BallotBoxes) Len() int {
	return len(b.Results)
}

func (b *BallotBoxes) HasMessage(m sebakcommon.Message) bool {
	return b.HasMessageByHash(m.GetHash())
}

func (b *BallotBoxes) HasMessageByHash(hash string) bool {
	_, ok := b.Results[hash]
	return ok
}

func (b *BallotBoxes) VotingResult(ballot Ballot) *VotingResult {
	if !b.HasMessageByHash(ballot.MessageHash()) {
		return nil
	}

	return b.Results[ballot.MessageHash()]
}

func (b *BallotBoxes) IsVoted(ballot Ballot) bool {
	vr := b.VotingResult(ballot)
	if vr == nil {
		return false
	}

	return vr.IsVoted(ballot)
}

func (b *BallotBoxes) AddVotingResult(vr *VotingResult, ballot Ballot) (err error) {
	b.Lock()
	defer b.Unlock()

	b.Results[vr.MessageHash] = vr

	if ballot.CanFitInVotingBox() {
		err = b.VotingBox.AddVotingResult(vr)
	} else if ballot.CanFitInWaitingBox() {
		err = b.WaitingBox.AddVotingResult(vr)
	} else {
		// do nothing
	}

	return
}

func (b *BallotBoxes) RemoveVotingResult(vr *VotingResult) (err error) {
	if !b.HasMessageByHash(vr.MessageHash) {
		err = sebakerror.ErrorVotingResultNotFound
		return
	}

	delete(b.Results, vr.MessageHash)
	delete(b.Messages, vr.MessageHash)

	return
}

func (b *BallotBoxes) AddBallot(ballot Ballot) (isNew bool, err error) {
	b.Lock()
	defer b.Unlock()

	var vr *VotingResult

	isNew = !b.HasMessageByHash(ballot.MessageHash())

	if !isNew {
		vr = b.VotingResult(ballot)
		if err = vr.Add(ballot); err != nil {
			return
		}
		if b.ReservedBox.HasMessageByHash(ballot.MessageHash()) {
			if err = b.ReservedBox.RemoveVotingResult(vr); err != nil {
				log.Error("ReservedBox has a message but cannot remove it", "MessageHash", ballot.MessageHash(), "error", err)
			}

			err = b.AddVotingResult(vr, ballot)
			if err != nil {
				log.Warn("failed to add VotingResult", "MessageHash", ballot.MessageHash(), "error", err)
				err = nil
			}
		}
		return
	}

	vr, err = NewVotingResult(ballot)
	if err != nil {
		return
	}

	// unknown ballot will be in `WaitingBox`
	if err = b.AddVotingResult(vr, ballot); err != nil {
		log.Warn("failed to add VotingResult", "MessageHash", ballot.MessageHash(), "error", err)
	}

	if _, found := b.Messages[ballot.MessageHash()]; !found {
		b.Messages[ballot.MessageHash()] = ballot.Data().Data.(sebakcommon.Message)
	}

	return
}

type BallotBox struct {
	sebakcommon.SafeLock

	Hashes map[string]bool // `Message.Hash`es
}

func NewBallotBox() *BallotBox {
	return &BallotBox{Hashes: make(map[string]bool)}
}

func (b *BallotBox) Len() int {
	return len(b.Hashes)
}

func (b *BallotBox) HasMessage(m sebakcommon.Message) bool {
	return b.HasMessageByHash(m.GetHash())
}

func (b *BallotBox) HasMessageByHash(hash string) bool {
	_, found := b.Hashes[hash]
	return found
}

func (b *BallotBox) AddVotingResult(vr *VotingResult) (err error) {
	if b.HasMessageByHash(vr.MessageHash) {
		err = sebakerror.ErrorVotingResultAlreadyExists
		return
	}

	b.Lock()
	defer b.Unlock()

	b.Hashes[vr.MessageHash] = true

	return
}

func (b *BallotBox) RemoveVotingResult(vr *VotingResult) (err error) {
	if !b.HasMessageByHash(vr.MessageHash) {
		err = sebakerror.ErrorVotingResultNotFound
		return
	}

	b.Lock()
	defer b.Unlock()

	delete(b.Hashes, vr.MessageHash)

	return
}
