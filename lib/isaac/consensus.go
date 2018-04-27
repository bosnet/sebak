package isaac

import (
	"sort"

	"github.com/spikeekips/sebak/lib/error"
	"github.com/spikeekips/sebak/lib/storage"
	"github.com/spikeekips/sebak/lib/transport"
	"github.com/spikeekips/sebak/lib/util"
)

type ISAAC struct {
	util.SafeLock

	Node                  Node
	VotingThresholdPolicy VotingThresholdPolicy

	Storage   *storage.LevelDBBackend
	Transport transport.Transport

	Boxes *BallotBoxes
}

func NewISAAC(node Node, votingThresholdPolicy VotingThresholdPolicy, st *storage.LevelDBBackend, tp transport.Transport) (is *ISAAC, err error) {
	is = &ISAAC{
		Node: node,
		VotingThresholdPolicy: votingThresholdPolicy,
		Boxes: NewBallotBoxes(),

		Storage:   st,
		Transport: tp,
	}

	return
}

func (is *ISAAC) ReceiveMessage(m util.Message) (ballot Ballot, err error) {
	/*
		Previously the new incoming Message must be checked,
			- TODO `Message` must be saved in `BlockTransactionHistory`
			- TODO check already `IsWellFormed()`
			- TODO check already in BlockTransaction
			- TODO check already in BlockTransactionHistory
	*/

	if is.Boxes.HasMessage(m) {
		err = sebak_error.ErrorNewButKnownMessage
		return
	}

	if ballot, err = NewBallotFromMessage(is.Node.GetKeypair().Address(), m); err != nil {
		return
	}

	// self-sign; make new `Ballot` from `Message`
	ballot.SetState(BallotStateINIT)
	ballot.Vote(VotingYES) // TODO YES or NO
	ballot.UpdateHash()
	ballot.Sign(is.Node.GetKeypair())

	if err = ballot.IsWellFormed(); err != nil {
		return
	}

	if _, err = is.Boxes.AddBallot(ballot); err != nil {
		return
	}

	return
}

func (is *ISAAC) ReceiveBallot(ballot Ballot) (vr *VotingResult, err error) {
	/*
		TODO Previously the new incoming Ballot must be checked `IsWellFormed()`
	*/

	switch ballot.GetState() {
	case BallotStateINIT:
		vr, err = is.receiveBallotStateINIT(ballot)
	case BallotStateALLCONFIRM:
		err = sebak_error.ErrorBallotHasInvalidState
		return
	default:
		vr, err = is.receiveBallotVotingStates(ballot)
	}

	if err != nil {
		return
	}

	return
}

func (is *ISAAC) receiveBallotStateINIT(ballot Ballot) (vr *VotingResult, err error) {
	var isNew bool

	if isNew, err = is.Boxes.AddBallot(ballot); err != nil {
		return
	}

	if !isNew {
		// check `VotingResult.CanGetResult()`; if possible, `VotingResult.GetResult()`
		vrt := is.Boxes.GetVotingResult(ballot)
		if !vrt.CanGetResult(is.VotingThresholdPolicy) {
			return
		}

		state, passed := vrt.GetResult(is.VotingThresholdPolicy)
		if !passed || state.Next() < vrt.State {
			return
		}

		// TODO move to next state, so broadcast new ballot
		if !vrt.SetState(state.Next()) {
			err = sebak_error.ErrorVotingResultFailedToSetState
			return
		}
		is.Boxes.waitingToVoting(vrt)

		vr = vrt

		return
	}

	var newBallot Ballot
	if newBallot, err = NewBallotFromMessage(is.Node.GetKeypair().Address(), ballot.GetMessage()); err != nil {
		return
	}

	// self-sign
	newBallot.SetState(BallotStateINIT)
	newBallot.Vote(VotingYES) // TODO YES or NO
	newBallot.UpdateHash()
	newBallot.Sign(is.Node.GetKeypair())

	if err = newBallot.IsWellFormed(); err != nil {
		return
	}

	if _, err = is.Boxes.AddBallot(newBallot); err != nil {
		return
	}

	// TODO this ballot should be broadcasted

	return
}

func (is *ISAAC) receiveBallotVotingStates(ballot Ballot) (vr *VotingResult, err error) {
	// FIXME if ballot indicates the unknown messsage, ignore it
	if !is.Boxes.HasMessage(ballot.GetMessage()) {
		return
	}

	if _, err = is.Boxes.AddBallot(ballot); err != nil {
		return
	}

	if !is.Boxes.VotingBox.HasMessage(ballot.GetMessage()) {
		return
	}

	vrt := is.Boxes.GetVotingResult(ballot)
	if !vrt.CanGetResult(is.VotingThresholdPolicy) {
		return
	}

	state, passed := vrt.GetResult(is.VotingThresholdPolicy)
	if !passed || state.Next() < vrt.State {
		return
	}

	// TODO move to next state, so broadcast new ballot
	if !vrt.SetState(state.Next()) {
		err = sebak_error.ErrorVotingResultFailedToSetState
		return
	}

	// TODO if state reaches `BallotStateALLCONFIRM`, externalize it.

	vr = vrt

	return
}

type BallotBoxes struct {
	util.SafeLock

	Results map[ /* `Message.GetHash()`*/ string]*VotingResult

	WaitingBox  *BallotBox
	VotingBox   *BallotBox
	ReservedBox *BallotBox
}

func NewBallotBoxes() *BallotBoxes {
	return &BallotBoxes{
		Results:     map[string]*VotingResult{},
		WaitingBox:  NewBallotBox(),
		VotingBox:   NewBallotBox(),
		ReservedBox: NewBallotBox(),
	}
}

func (b *BallotBoxes) Len() int {
	return len(b.Results)
}

func (b *BallotBoxes) HasMessage(m util.Message) bool {
	return b.HasMessageByString(m.GetHash())
}

func (b *BallotBoxes) HasMessageByString(hash string) bool {
	_, ok := b.Results[hash]
	return ok
}

func (b *BallotBoxes) GetVotingResult(ballot Ballot) *VotingResult {
	if !b.HasMessage(ballot.GetMessage()) {
		return nil
	}

	return b.Results[ballot.GetMessage().GetHash()]
}

func (b *BallotBoxes) IsVoted(ballot Ballot) bool {
	vr := b.GetVotingResult(ballot)
	if vr == nil {
		return false
	}

	return vr.IsVoted(ballot)
}

func (b *BallotBoxes) AddVotingResult(vr *VotingResult, bb *BallotBox) (err error) {
	b.Lock()
	defer b.Unlock()

	b.Results[vr.MessageHash] = vr
	bb.AddVotingResult(vr) // TODO detect error

	return
}

func (b *BallotBoxes) RemoveVotingResult(vr *VotingResult, bb *BallotBox) (err error) {
	if !b.HasMessageByString(vr.MessageHash) {
		err = sebak_error.ErrorVotingResultNotFound
		return
	}

	b.Lock()
	defer b.Unlock()

	delete(b.Results, vr.MessageHash)

	bb.RemoveVotingResult(vr) // TODO detect error

	return
}

func (b *BallotBoxes) AddBallot(ballot Ballot) (isNew bool, err error) {
	b.Lock()
	defer b.Unlock()

	var vr *VotingResult

	isNew = !b.HasMessage(ballot.GetMessage())
	if !isNew {
		vr = b.GetVotingResult(ballot)
		if err = vr.Add(ballot); err != nil {
			return
		}

		if b.ReservedBox.HasMessage(ballot.GetMessage()) {
			b.reserveToVoting(vr)
		}
		return
	}

	vr, err = NewVotingResult(ballot)
	if err != nil {
		return
	}

	// unknown ballot will be in `WaitingBox`
	err = b.AddVotingResult(vr, b.WaitingBox)

	return
}

func (b *BallotBoxes) reserveToVoting(vr *VotingResult) (err error) {
	b.ReservedBox.RemoveVotingResult(vr) // detect error
	b.VotingBox.AddVotingResult(vr)      // detect error

	return
}

func (b *BallotBoxes) waitingToVoting(vr *VotingResult) (err error) {
	b.WaitingBox.RemoveVotingResult(vr) // detect error
	b.VotingBox.AddVotingResult(vr)     // detect error

	return
}

type BallotBox struct {
	util.SafeLock

	Hashes sort.StringSlice // `Message.Hash`es
}

func NewBallotBox() *BallotBox {
	return &BallotBox{}
}

func (b *BallotBox) Len() int {
	return len(b.Hashes)
}

func (b *BallotBox) HasMessage(m util.Message) bool {
	return b.HasMessageByString(m.GetHash())
}

func (b *BallotBox) HasMessageByString(hash string) bool {
	l := len(b.Hashes)
	if l < 1 {
		return false
	}
	i := sort.SearchStrings(b.Hashes, hash)

	return i != l && b.Hashes[i] == hash
}

func (b *BallotBox) AddVotingResult(vr *VotingResult) (err error) {
	if b.HasMessageByString(vr.MessageHash) {
		err = sebak_error.ErrorVotingResultAlreadyExists
		return
	}

	b.Lock()
	defer b.Unlock()

	b.Hashes = append(b.Hashes, vr.MessageHash)
	sort.Strings(b.Hashes)

	return
}

func (b *BallotBox) RemoveVotingResult(vr *VotingResult) (err error) {
	if !b.HasMessageByString(vr.MessageHash) {
		err = sebak_error.ErrorVotingResultNotFound
		return
	}

	b.Lock()
	defer b.Unlock()

	i := sort.SearchStrings(b.Hashes, vr.MessageHash)
	b.Hashes = append(b.Hashes[:i], b.Hashes[i+1:]...)
	sort.Strings(b.Hashes)

	return
}
