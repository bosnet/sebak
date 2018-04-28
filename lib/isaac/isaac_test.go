package consensus

import (
	"encoding/json"
	"errors"
	"net"
	"testing"

	"github.com/btcsuite/btcutil/base58"
	"github.com/spikeekips/sebak/lib/error"
	"github.com/spikeekips/sebak/lib/storage"
	"github.com/spikeekips/sebak/lib/util"
	"github.com/stellar/go/keypair"
)

type DummyNode struct {
	keypair *keypair.Full
	alias   string
}

func (n DummyNode) GetKeypair() *keypair.Full {
	return n.keypair
}

func (n DummyNode) GetAlias() string {
	return n.alias
}

func NewNode(kp *keypair.Full, alias string) DummyNode {
	return DummyNode{
		keypair: kp,
		alias:   alias,
	}
}

func NewRandomNode() DummyNode {
	kp, _ := keypair.Random()
	return NewNode(kp, util.GenerateUUID())
}

type DummyTransport struct{}

func (t DummyTransport) Send(addr net.TCPAddr, b []byte) error {
	return nil
}

func (t DummyTransport) Receive() ([]byte, error) {
	return []byte{}, nil
}

type DummyMessage struct {
	Hash string
	Data string
}

func NewDummyMessage(data string) DummyMessage {
	d := DummyMessage{Data: data}
	d.UpdateHash()

	return d
}

func (m DummyMessage) IsWellFormed() error {
	return nil
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
	st, _ := storage.NewTestMemoryLevelDBBackend()
	tp := &DummyTransport{}

	policy, _ := NewDefaultVotingThresholdPolicy(100, 30, 30)
	policy.SetValidators(uint64(minimumValidators))

	is, _ := NewISAAC(NewRandomNode(), policy, st, tp)

	return is
}

func makeBallot(kp *keypair.Full, m util.Message, state BallotState) Ballot {
	ballot, _ := NewBallotFromMessage(kp.Address(), m)
	ballot.SetState(state)
	ballot.Vote(VotingYES)
	ballot.UpdateHash()
	ballot.Sign(kp)

	return ballot
}

func TestNewISAAC(t *testing.T) {
	st, _ := storage.NewTestMemoryLevelDBBackend()
	tp := &DummyTransport{}

	policy, _ := NewDefaultVotingThresholdPolicy(100, 30, 30)
	policy.SetValidators(uint64(1))

	is, err := NewISAAC(NewRandomNode(), policy, st, tp)
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
		if _, err = is.ReceiveMessage(m); err != sebak_error.ErrorNewButKnownMessage {
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

	var numberOfBallots uint64 = 5

	m := NewDummyMessage(util.GenerateUUID())

	// make ballots
	var err error
	var ballots []Ballot
	var vr *VotingResult
	for i := 0; i < int(numberOfBallots); i++ {
		kp, _ := keypair.Random()

		ballot := makeBallot(kp, m, BallotStateINIT)
		ballots = append(ballots, ballot)

		if vr, err = is.ReceiveBallot(ballot); err != nil {
			t.Error(err)
			return
		}
		if vr != nil {
			break
		}

		if !is.Boxes.IsVoted(ballot) {
			t.Error("failed to vote")
			return
		}
	}

	if vr == nil {
		t.Error("failed to get result")
		return
	}
}

func TestISAACReceiveBallotStateINITAndVotingBox(t *testing.T) {
	is := makeISAAC(5)

	var numberOfBallots uint64 = 5

	m := NewDummyMessage(util.GenerateUUID())

	// make ballots
	var err error
	var vr *VotingResult
	var ballots []Ballot
	for i := 0; i < int(numberOfBallots); i++ {
		kp, _ := keypair.Random()

		ballot := makeBallot(kp, m, BallotStateINIT)
		ballots = append(ballots, ballot)

		if vr, err = is.ReceiveBallot(ballot); err != nil {
			t.Error(err)
			return
		}
		if vr != nil {
			break
		}
		if !is.Boxes.IsVoted(ballot) {
			t.Error("failed to vote")
			return
		}
	}

	if vr == nil {
		t.Error("failed to get result")
		return
	}

	if is.Boxes.WaitingBox.HasMessage(ballots[0].GetMessage()) {
		t.Error("after `INIT`, the ballot must move to `VotingBox`")
	}
}

func voteISAACReceiveBallot(is *ISAAC, ballots []Ballot, kps []*keypair.Full, state BallotState) (vr *VotingResult, err error) {
	var vrt *VotingResult
	for i, ballot := range ballots {
		ballot.SetState(state)
		ballot.UpdateHash()
		ballot.Sign(kps[i])

		if vrt, err = is.ReceiveBallot(ballot); err != nil {
			break
		}
		if vrt != nil {
			vr = vrt
		}
		if !is.Boxes.IsVoted(ballot) {
			return
		}
	}
	if err != nil {
		return
	}

	return
}

func TestISAACReceiveBallotStateTransition(t *testing.T) {
	var numberOfBallots uint64 = 5
	var minimumValidators int = 3 // must be passed

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
		vr, err := voteISAACReceiveBallot(is, ballots, kps, BallotStateINIT)
		if err != nil {
			t.Error(err)
			return
		}

		if is.Boxes.WaitingBox.HasMessage(ballots[0].GetMessage()) {
			t.Error("after `INIT`, the ballot must move to `VotingBox`")
		}

		if vr == nil {
			err = errors.New("failed to get result")
			return
		}
		if vr.State != BallotStateSIGN {
			err = errors.New("`VotingResult.State` must be `BallotStateSIGN`")
			return
		}

		if !is.Boxes.VotingBox.HasMessage(ballots[0].GetMessage()) {
			err = errors.New("after `INIT`, the ballot must move to `VotingBox`")
			return
		}
	}

	// SIGN -> ACCEPT
	{
		vr, err := voteISAACReceiveBallot(is, ballots, kps, BallotStateSIGN)
		if err != nil {
			t.Error(err)
			return
		}

		if vr == nil {
			err = errors.New("failed to get result")
			return
		}
		if vr.State != BallotStateACCEPT {
			err = errors.New("`VotingResult.State` must be `BallotStateACCEPT`")
			return
		}

		if !is.Boxes.VotingBox.HasMessage(ballots[0].GetMessage()) {
			err = errors.New("after `INIT`, the ballot must move to `VotingBox`")
			return
		}
	}

	// ACCEPT -> ALL-CONFIRM
	{
		vr, err := voteISAACReceiveBallot(is, ballots, kps, BallotStateACCEPT)
		if err != nil {
			t.Error(err)
			return
		}
		if vr == nil {
			err = errors.New("failed to get result")
			return
		}
		if vr.State != BallotStateALLCONFIRM {
			err = errors.New("`VotingResult.State` must be `BallotStateALLCONFIRM`")
			return
		}

		if !is.Boxes.VotingBox.HasMessage(ballots[0].GetMessage()) {
			err = errors.New("after `INIT`, the ballot must move to `VotingBox`")
			return
		}
	}
}

func TestISAACReceiveSameBallotStates(t *testing.T) {
	var numberOfBallots uint64 = 5
	var minimumValidators int = 3

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
		vr, err := voteISAACReceiveBallot(is, ballots, kps, BallotStateINIT)
		if err != nil {
			t.Error(err)
			return
		}

		if is.Boxes.WaitingBox.HasMessage(ballots[0].GetMessage()) {
			t.Error("after `INIT`, the ballot must move to `VotingBox`")
		}
		if vr == nil {
			t.Error("failed to get result")
			return
		}
		if vr.State != BallotStateSIGN {
			err = errors.New("`VotingResult.State` must be `BallotStateSIGN`")
		}

		if vr.GetVotedCount(BallotStateINIT) != int(numberOfBallots)+1 {
			t.Error("some ballot was not voted")
			return
		}

		if vr.GetVotedCount(BallotStateSIGN) != 0 || vr.GetVotedCount(BallotStateACCEPT) != 0 || vr.GetVotedCount(BallotStateALLCONFIRM) != 0 {
			t.Error("unexpected ballots found")
			return
		}
	}

	vrFirst := is.Boxes.GetVotingResult(ballots[0])
	{
		vr, err := voteISAACReceiveBallot(is, ballots, kps, BallotStateINIT)
		if err != nil {
			t.Error(err)
			return
		}
		if vr != nil {
			t.Error("already state was changed to `BallotStateSIGN`")
			return
		}
	}
	vrSecond := is.Boxes.GetVotingResult(ballots[0])
	if vrSecond.GetVotedCount(BallotStateINIT) != int(numberOfBallots)+1 {
		t.Error("some ballot was not voted")
		return
	}

	if vrSecond.GetVotedCount(BallotStateSIGN) != 0 || vrSecond.GetVotedCount(BallotStateACCEPT) != 0 || vrSecond.GetVotedCount(BallotStateALLCONFIRM) != 0 {
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
