package sebak

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/btcsuite/btcutil/base58"
	"github.com/stellar/go/keypair"

	"github.com/spikeekips/sebak/lib/error"
	"github.com/spikeekips/sebak/lib/util"
)

func NewRandomNode() util.Node {
	kp, _ := keypair.Random()
	a, _ := util.NewValidator(kp.Address(), &util.Endpoint{}, "")
	a.SetKeypair(kp)
	return a
}

type DummyMessage struct {
	T    string
	Hash string
	Data string
}

func NewDummyMessage(data string) DummyMessage {
	d := DummyMessage{T: "dummy-message", Data: data}
	d.UpdateHash()

	return d
}

func (m DummyMessage) IsWellFormed() error {
	return nil
}

func (m DummyMessage) GetType() string {
	return m.T
}

func (m DummyMessage) Equal(n util.Message) bool {
	return m.Hash == n.GetHash()
}

func (m DummyMessage) GetHash() string {
	return m.Hash
}

func (m *DummyMessage) UpdateHash() {
	m.Hash = base58.Encode(util.MustMakeObjectHash(m.Data))
}

func (m DummyMessage) Serialize() ([]byte, error) {
	return json.Marshal(m)
}

func (m DummyMessage) String() string {
	s, _ := json.MarshalIndent(m, "  ", " ")
	return string(s)
}

func makeISAAC(minimumValidators int) *ISAAC {
	policy, _ := NewDefaultVotingThresholdPolicy(100, 30, 30)
	policy.SetValidators(minimumValidators)

	is, _ := NewISAAC(NewRandomNode(), policy)

	return is
}

func makeBallot(kp *keypair.Full, m util.Message, state BallotState) Ballot {
	ballot, _ := NewBallotFromMessage(kp.Address(), m)
	ballot.SetState(state)
	ballot.Vote(VotingYES)
	ballot.Sign(kp)

	return ballot
}

func TestNewISAAC(t *testing.T) {
	policy, _ := NewDefaultVotingThresholdPolicy(100, 30, 30)
	policy.SetValidators(1)

	is, err := NewISAAC(NewRandomNode(), policy)
	if err != nil {
		t.Errorf("`NewISAAC` must not be failed: %v", err)
		return
	}

	// check BallotBox is empty
	if is.Boxes.WaitingBox.Len() > 0 {
		t.Error("`WaitingBox` must be empty")
		return
	}
	if is.Boxes.VotingBox.Len() > 0 {
		t.Error("`VotingBox` must be empty")
		return
	}
	if is.Boxes.ReservedBox.Len() > 0 {
		t.Error("`ReservedBox` must be empty")
		return
	}
}

func TestISAACNewIncomingMessage(t *testing.T) {
	is := makeISAAC(1)

	m := NewDummyMessage(util.GenerateUUID())

	{
		var err error
		if _, err = is.ReceiveMessage(m); err != nil {
			t.Error(err)
			return
		}
		if !is.Boxes.HasMessage(m) {
			t.Error("failed to add message")
			return
		}
		if !is.Boxes.WaitingBox.HasMessage(m) {
			t.Error("failed to add message to `WaitingBox`")
			return
		}
	}

	// receive same message
	{
		var err error
		if _, err = is.ReceiveMessage(m); err != sebakerror.ErrorNewButKnownMessage {
			t.Error("incoming known message must occurr `ErrorNewButKnownMessage`")
			return
		}
		if !is.Boxes.HasMessage(m) {
			t.Error("failed to find message")
			return
		}
		if !is.Boxes.WaitingBox.HasMessage(m) {
			t.Error("failed to find message to `WaitingBox`")
			return
		}

		if is.Boxes.WaitingBox.Len() != 1 {
			t.Error("`WaitingBox` has another `Message`")
		}
		if is.Boxes.VotingBox.Len() > 0 {
			t.Error("`VotingBox` must be empty")
		}
		if is.Boxes.ReservedBox.Len() > 0 {
			t.Error("`ReservedBox` must be empty")
		}
	}

	// send another message
	{
		var err error

		another := NewDummyMessage(util.GenerateUUID())

		_, err = is.ReceiveMessage(another)
		if err != nil {
			t.Errorf("failed to add another message: %v", err)
			return
		}
		if !is.Boxes.HasMessage(another) {
			t.Error("failed to find message")
			return
		}
		if !is.Boxes.WaitingBox.HasMessage(another) {
			t.Error("failed to find message to `WaitingBox`")
			return
		}

		if is.Boxes.WaitingBox.Len() != 2 {
			t.Error("`WaitingBox` failed to add another")
		}
		if is.Boxes.VotingBox.Len() > 0 {
			t.Error("`VotingBox` must be empty")
		}
		if is.Boxes.ReservedBox.Len() > 0 {
			t.Error("`ReservedBox` must be empty")
		}
	}

}

func TestISAACReceiveBallotStateINIT(t *testing.T) {
	is := makeISAAC(1)
	m := NewDummyMessage(util.GenerateUUID())

	kp, _ := keypair.Random()
	ballot := makeBallot(kp, m, BallotStateINIT)

	// new ballot from another node
	if _, err := is.ReceiveBallot(ballot); err != nil {
		t.Error(err)
		return
	}

	if !is.Boxes.IsVoted(ballot) {
		t.Error("failed to vote")
		return
	}
}

func TestISAACIsVoted(t *testing.T) {
	is := makeISAAC(1)
	m := NewDummyMessage(util.GenerateUUID())

	is.ReceiveMessage(m)

	kp, _ := keypair.Random()

	ballot := makeBallot(kp, m, BallotStateINIT)

	if is.Boxes.IsVoted(ballot) {
		t.Error("`IsVoted` must be `false` ")
		return
	}

	is.ReceiveBallot(ballot)
	if !is.Boxes.IsVoted(ballot) {
		t.Error("failed to vote")
		return
	}
}

func TestISAACReceiveBallotStateINITAndMoveNextState(t *testing.T) {
	is := makeISAAC(5)

	var numberOfBallots int = 5

	m := NewDummyMessage(util.GenerateUUID())

	// make ballots
	var err error
	var ballots []Ballot
	var vs VotingStateStaging

	for i := 0; i < int(numberOfBallots); i++ {
		kp, _ := keypair.Random()

		ballot := makeBallot(kp, m, BallotStateINIT)
		ballots = append(ballots, ballot)

		if vs, err = is.ReceiveBallot(ballot); err != nil {
			t.Error(err)
			return
		}

		if !is.Boxes.IsVoted(ballot) {
			t.Error("failed to vote")
			return
		}
	}

	if vs.IsClosed() {
		t.Error("just state changed, not `VotingResult` closed")
		return
	}

	vs = is.Boxes.VotingResult(ballots[0]).LatestStaging()
	if vs.IsEmpty() {
		t.Error("failed to get valid `VotingStateStaging`")
		return
	}
	if !vs.IsChanged() {
		t.Error("failed to change state")
		return
	}
}

func TestISAACReceiveBallotStateINITAndVotingBox(t *testing.T) {
	is := makeISAAC(5)

	var numberOfBallots int = 5

	m := NewDummyMessage(util.GenerateUUID())

	// make ballots
	var err error
	var vs VotingStateStaging
	var ballots []Ballot
	for i := 0; i < int(numberOfBallots); i++ {
		kp, _ := keypair.Random()

		ballot := makeBallot(kp, m, BallotStateINIT)
		ballots = append(ballots, ballot)

		if vs, err = is.ReceiveBallot(ballot); err != nil {
			t.Error(err)
			return
		}
		if !is.Boxes.IsVoted(ballot) {
			t.Error("failed to vote")
			return
		}
	}

	vs = is.Boxes.VotingResult(ballots[0]).LatestStaging()
	if !vs.IsChanged() {
		t.Error("failed to get result")
		return
	}

	if is.Boxes.WaitingBox.HasMessageByString(ballots[0].MessageHash()) {
		t.Error("after `INIT`, the ballot must move to `VotingBox`")
	}
}

func voteISAACReceiveBallot(is *ISAAC, ballots []Ballot, kps []*keypair.Full, state BallotState) (vs VotingStateStaging, err error) {
	for i, ballot := range ballots {
		ballot.SetState(state)
		ballot.Sign(kps[i])

		if vs, err = is.ReceiveBallot(ballot); err != nil {
			break
		}
		if !is.Boxes.IsVoted(ballot) {
			return
		}
	}
	if err != nil {
		return
	}

	vs = is.Boxes.VotingResult(ballots[0]).LatestStaging()

	return
}

func TestISAACReceiveBallotStateTransition(t *testing.T) {
	var numberOfBallots int = 5
	var minimumValidators = 3 // must be passed

	is := makeISAAC(minimumValidators)

	m := NewDummyMessage(util.GenerateUUID())

	var ballots []Ballot
	var kps []*keypair.Full

	for i := 0; i < int(numberOfBallots); i++ {
		kp, _ := keypair.Random()
		kps = append(kps, kp)

		ballots = append(ballots, makeBallot(kp, m, BallotStateINIT))
	}

	// INIT -> SIGN
	{
		vs, err := voteISAACReceiveBallot(is, ballots, kps, BallotStateINIT)
		if err != nil {
			t.Error(err)
			return
		}

		if is.Boxes.WaitingBox.HasMessageByString(ballots[0].MessageHash()) {
			t.Error("after `INIT`, the ballot must move to `VotingBox`")
		}

		if vs.IsEmpty() {
			err = errors.New("failed to get result")
			return
		}
		if vs.State != BallotStateSIGN {
			err = errors.New("`VotingResult.State` must be `BallotStateSIGN`")
			return
		}

		if !is.Boxes.VotingBox.HasMessageByString(ballots[0].MessageHash()) {
			err = errors.New("after `INIT`, the ballot must move to `VotingBox`")
			return
		}
	}

	// SIGN -> ACCEPT
	{
		vs, err := voteISAACReceiveBallot(is, ballots, kps, BallotStateSIGN)
		if err != nil {
			t.Error(err)
			return
		}

		if vs.IsEmpty() {
			err = errors.New("failed to get result")
			return
		}
		if vs.State != BallotStateACCEPT {
			err = errors.New("`VotingResult.State` must be `BallotStateACCEPT`")
			return
		}

		if !is.Boxes.VotingBox.HasMessageByString(ballots[0].MessageHash()) {
			err = errors.New("after `INIT`, the ballot must move to `VotingBox`")
			return
		}
	}

	// ACCEPT -> ALL-CONFIRM
	{
		vs, err := voteISAACReceiveBallot(is, ballots, kps, BallotStateACCEPT)
		if err != nil {
			t.Error(err)
			return
		}
		if vs.IsEmpty() {
			err = errors.New("failed to get result")
			return
		}
		if vs.State != BallotStateALLCONFIRM {
			err = errors.New("`VotingResult.State` must be `BallotStateALLCONFIRM`")
			return
		}

		if !is.Boxes.VotingBox.HasMessageByString(ballots[0].MessageHash()) {
			err = errors.New("after `INIT`, the ballot must move to `VotingBox`")
			return
		}
	}
}

func TestISAACReceiveSameBallotStates(t *testing.T) {
	var numberOfBallots int = 5
	var minimumValidators = 3

	is := makeISAAC(minimumValidators)

	m := NewDummyMessage(util.GenerateUUID())

	var ballots []Ballot
	var kps []*keypair.Full

	for i := 0; i < int(numberOfBallots); i++ {
		kp, _ := keypair.Random()
		kps = append(kps, kp)

		ballots = append(ballots, makeBallot(kp, m, BallotStateINIT))
	}

	{
		vs, err := voteISAACReceiveBallot(is, ballots, kps, BallotStateINIT)
		if err != nil {
			t.Error(err)
			return
		}

		if is.Boxes.WaitingBox.HasMessageByString(ballots[0].MessageHash()) {
			t.Error("after `INIT`, the ballot must move to `VotingBox`")
		}
		if vs.IsEmpty() {
			t.Error("failed to get result")
			return
		}
		if vs.State != BallotStateSIGN {
			err = errors.New("`VotingResult.State` must be `BallotStateSIGN`")
		}

		vr := is.Boxes.VotingResult(ballots[0])
		if vr.VotedCount(BallotStateINIT) != int(numberOfBallots)+1 {
			t.Error("some ballot was not voted")
			return
		}

		if vr.VotedCount(BallotStateSIGN) != 0 || vr.VotedCount(BallotStateACCEPT) != 0 || vr.VotedCount(BallotStateALLCONFIRM) != 0 {
			t.Error("unexpected ballots found")
			return
		}
	}

	vrFirst := is.Boxes.VotingResult(ballots[0])
	{
		_, err := voteISAACReceiveBallot(is, ballots, kps, BallotStateINIT)
		if err != nil {
			t.Error(err)
			return
		}
	}
	vrSecond := is.Boxes.VotingResult(ballots[0])
	if vrSecond.VotedCount(BallotStateINIT) != int(numberOfBallots)+1 {
		t.Error("some ballot was not voted")
		return
	}

	if vrSecond.VotedCount(BallotStateSIGN) != 0 || vrSecond.VotedCount(BallotStateACCEPT) != 0 || vrSecond.VotedCount(BallotStateALLCONFIRM) != 0 {
		t.Error("unexpected ballots found")
		return
	}

	for k, v := range vrFirst.Ballots {
		for k0, v0 := range v {
			if v0.Hash != vrSecond.Ballots[k][k0].Hash {
				t.Error("not matched")
				break
			}
		}
	}
}
