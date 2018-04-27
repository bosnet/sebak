package isaac

import (
	"encoding/json"
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

func makeISAAC(validators int) *ISAAC {
	st, _ := storage.NewTestMemoryLevelDBBackend()
	tp := &DummyTransport{}

	policy, _ := NewDefaultVotingThresholdPolicy(100, 30, 30)
	policy.SetValidators(uint64(validators))

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

func TestISAACReceiveBallotStateTransition(t *testing.T) {
	var numberOfBallots uint64 = 5
	var validators int = 3

	is := makeISAAC(validators)

	m := NewDummyMessage(util.GenerateUUID())

	var ballots []Ballot
	var kps []*keypair.Full

	// INIT -> SIGN
	{
		var err error
		var vr *VotingResult
		for i := 0; i < int(numberOfBallots); i++ {
			kp, _ := keypair.Random()
			kps = append(kps, kp)

			ballot := makeBallot(kp, m, BallotStateINIT)
			ballots = append(ballots, ballot)

			var vrt *VotingResult
			if vrt, err = is.ReceiveBallot(ballot); err != nil {
				t.Error(err)
				return
			}
			if vr == nil && vrt != nil {
				vr = vrt
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
		if vr.State != BallotStateSIGN {
			t.Error("`VotingResult.State` must be `BallotStateSIGN`")
			return
		}

		if is.Boxes.WaitingBox.HasMessage(ballots[0].GetMessage()) {
			t.Error("after `INIT`, the ballot must move to `VotingBox`")
		}
		if !is.Boxes.VotingBox.HasMessage(ballots[0].GetMessage()) {
			t.Error("after `INIT`, the ballot must move to `VotingBox`")
		}
	}

	// SIGN -> ACCEPT
	{
		var err error
		var vr *VotingResult
		for i, ballot := range ballots {
			ballot.SetState(ballot.GetState().Next())
			ballot.UpdateHash()
			ballot.Sign(kps[i])

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
		if vr.State != BallotStateACCEPT {
			t.Error("`VotingResult.State` must be `BallotStateACCEPT`")
			return
		}
	}

	{
		var err error
		var vr *VotingResult

		for i, ballot := range ballots {
			ballot.SetState(ballot.GetState().Next().Next())
			ballot.UpdateHash()
			ballot.Sign(kps[i])

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

		if vr.State != BallotStateALLCONFIRM {
			t.Error("`VotingResult.State` must be `BallotStateALLCONFIRM`")
			return
		}
	}
}
