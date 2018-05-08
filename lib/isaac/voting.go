package consensus

import (
	"encoding/json"
	"math"
	"sync"

	"github.com/spikeekips/sebak/lib/error"
	"github.com/spikeekips/sebak/lib/util"
)

type VotingHole string

const (
	VotingNOTYET VotingHole = "NOT-YET"
	VotingYES    VotingHole = "YES"
	VotingNO     VotingHole = "NO"
)

type VotingResultBallot struct {
	Hash       string // `Ballot.GetHash()`
	State      BallotState
	VotingHole VotingHole
	Reason     string
}

func NewVotingResultBallotFromBallot(ballot Ballot) VotingResultBallot {
	return VotingResultBallot{
		Hash:       ballot.GetHash(),
		State:      ballot.B.State,
		VotingHole: ballot.B.VotingHole,
		Reason:     ballot.B.Reason,
	}
}

type VotingResultBallots map[ /* NodeKey */ string]VotingResultBallot

// VotingStateStaging will keep the snapshot at changing state.
type VotingStateStaging struct {
	State         BallotState
	PreviousState BallotState

	ID          string     // ID is unique and sequenital
	MessageHash string     // MessageHash is `Message.Hash`
	VotingHole  VotingHole // voting is closed and it's last `VotingHole`
	Reason      error      // if `VotingNO` is concluded, the reason

	Ballots map[ /* NodeKey */ string]VotingResultBallot
}

func (vs VotingStateStaging) String() string {
	encoded, _ := json.MarshalIndent(vs, "", "  ")
	return string(encoded)
}

func (vs VotingStateStaging) IsChanged() bool {
	return vs.State > vs.PreviousState
}

func (vs VotingStateStaging) IsEmpty() bool {
	return len(vs.Ballots) < 1
}

func (vs VotingStateStaging) IsClosed() bool {
	if !vs.IsEmpty() {
		return false
	}
	if vs.VotingHole == VotingNO {
		return true
	}
	if vs.State == BallotStateALLCONFIRM {
		return true
	}

	return false
}

func (vs VotingStateStaging) IsStorable() bool {
	if !vs.IsClosed() {
		return false
	}
	if vs.State != BallotStateALLCONFIRM {
		return false
	}
	if vs.VotingHole == VotingNO {
		return false
	}

	return true
}

type VotingResult struct {
	sync.Mutex

	ID          string      // ID is unique and sequenital
	MessageHash string      // MessageHash is `Message.Hash`
	State       BallotState // Latest `BallotState`
	Ballots     map[BallotState]VotingResultBallots
	Staging     []VotingStateStaging // state changing histories
}

func NewVotingResult(ballot Ballot) (vr *VotingResult, err error) {
	ballots := map[BallotState]VotingResultBallots{
		BallotStateNONE:       VotingResultBallots{},
		BallotStateINIT:       VotingResultBallots{},
		BallotStateSIGN:       VotingResultBallots{},
		BallotStateACCEPT:     VotingResultBallots{},
		BallotStateALLCONFIRM: VotingResultBallots{},
	}

	ballots[ballot.State()][ballot.B.NodeKey] = NewVotingResultBallotFromBallot(ballot)

	vr = &VotingResult{
		ID:          util.GetUniqueIDFromUUID(),
		MessageHash: ballot.Message().GetHash(),
		State:       ballot.State(),
		Ballots:     ballots,
	}

	return
}

func (vr *VotingResult) IsClosed() bool {
	return vr.LatestStaging().IsClosed()
}

func (vr *VotingResult) SetState(state BallotState) bool {
	if vr.State >= state {
		return false
	}

	vr.Lock()
	defer vr.Unlock()

	vr.State = state

	return true
}

func (vr *VotingResult) Serialize() (encoded []byte, err error) {
	encoded, err = json.Marshal(vr)
	return
}

func (vr *VotingResult) String() string {
	encoded, _ := json.MarshalIndent(vr, "", "  ")
	return string(encoded)
}

func (vr *VotingResult) IsVoted(ballot Ballot) bool {
	ballots, ok := vr.Ballots[ballot.State()]
	if !ok {
		return false
	}
	if _, ok := ballots[ballot.B.NodeKey]; !ok {
		return false
	}

	return true
}

func (vr *VotingResult) VotedBallotsByState(state BallotState) VotingResultBallots {
	return vr.Ballots[state]
}

func (vr *VotingResult) VotedCount(state BallotState) int {
	return len(vr.VotedBallotsByState(state))
}

var VotingResultCheckerFuns = []util.CheckerFunc{
	checkBallotResultValidHash,
}

func (vr *VotingResult) Add(ballot Ballot) (err error) {
	vr.Lock()
	defer vr.Unlock()

	if err = util.Checker(VotingResultCheckerFuns...)(vr, ballot); err != nil {
		return
	}
	vr.Ballots[ballot.State()][ballot.B.NodeKey] = NewVotingResultBallotFromBallot(ballot)

	return
}

func (vr *VotingResult) CanCheckThreshold(state BallotState, threshold uint32) bool {
	if threshold < 1 {
		return false
	}
	if state == BallotStateNONE {
		return false
	}
	if vr.VotedCount(state) < int(threshold) {
		return false
	}

	return true
}

func (vr *VotingResult) CheckThreshold(state BallotState, threshold uint32) (VotingHole, bool) {
	if threshold < 1 {
		return VotingNOTYET, false
	}
	if state == BallotStateNONE {
		return VotingNOTYET, false
	}
	if vr.VotedCount(state) < int(threshold) {
		return VotingNOTYET, false
	}

	var yes int
	var no int
	for _, vrb := range vr.VotedBallotsByState(state) {
		if vrb.VotingHole == VotingYES {
			yes++
		} else if vrb.VotingHole == VotingNO {
			no++
		}
	}
	if yes >= int(threshold) {
		return VotingYES, true
	} else if no >= int(threshold) {
		return VotingNO, true
	}

	return VotingNOTYET, false
}

var CheckVotingThresholdSequence = []BallotState{
	BallotStateACCEPT,
	BallotStateSIGN,
	BallotStateINIT,
}

func (vr *VotingResult) MakeResult(policy VotingThresholdPolicy) (VotingHole, BallotState, bool) {
	if vr.State == BallotStateALLCONFIRM {
		return VotingNOTYET, BallotStateALLCONFIRM, false
	}

	for _, state := range CheckVotingThresholdSequence {
		if state < vr.State {
			break
		}

		t := policy.Threshold(state)
		if t < 1 {
			continue
		}
		votingHole, ended := vr.CheckThreshold(state, t)
		if ended {
			return votingHole, state, true
		}
	}

	return VotingNOTYET, vr.State, false
}

func (vr *VotingResult) ChangeState(votingHole VotingHole, state BallotState) (vs VotingStateStaging, err error) {
	previousState := vr.State
	if !vr.SetState(state.Next()) {
		err = sebakerror.ErrorVotingResultFailedToSetState
		return
	}

	vr.Lock()
	defer vr.Unlock()

	vs = vr.MakeStaging(votingHole, previousState, vr.State, state)
	vr.Staging = append(vr.Staging, vs)

	return
}

func (vr *VotingResult) MakeStaging(votingHole VotingHole, previousState, nextState, votingState BallotState) VotingStateStaging {
	// TODO set `VotingResult.Reason`
	return VotingStateStaging{
		ID:            vr.ID,
		MessageHash:   vr.MessageHash,
		State:         nextState,
		PreviousState: previousState,
		VotingHole:    votingHole,
		Ballots:       vr.VotedBallotsByState(votingState),
	}
}

func (vr *VotingResult) LatestStaging() VotingStateStaging {
	if len(vr.Staging) < 1 {
		return VotingStateStaging{}
	}
	return vr.Staging[len(vr.Staging)-1]
}

func (vr *VotingResult) CanGetResult(policy VotingThresholdPolicy) bool {
	if vr.State == BallotStateALLCONFIRM {
		return false
	}

	for _, state := range CheckVotingThresholdSequence {
		t := policy.Threshold(state)
		if t < 1 {
			continue
		}
		if vr.CanCheckThreshold(state, t) {
			return true
		}
	}

	return false
}

type VotingThresholdPolicy interface {
	Threshold(BallotState) uint32
	SetValidators(uint64) error
}

type DefaultVotingThresholdPolicy struct {
	init   uint32 // must be percentile
	sign   uint32
	accept uint32

	validators uint64
}

func (vt *DefaultVotingThresholdPolicy) Validators() uint64 {
	return vt.validators
}

func (vt *DefaultVotingThresholdPolicy) SetValidators(v uint64) error {
	if v < 1 {
		return sebakerror.ErrorVotingThresholdInvalidValidators
	}

	vt.validators = v

	return nil
}

func (vt *DefaultVotingThresholdPolicy) Threshold(state BallotState) uint32 {
	var t uint32
	switch state {
	case BallotStateINIT:
		t = vt.init
	case BallotStateSIGN:
		t = vt.sign
	case BallotStateACCEPT:
		t = vt.accept
	}

	v := float64(vt.validators) * (float64(t) / float64(100))
	return uint32(math.Ceil(v))
}

func NewDefaultVotingThresholdPolicy(init, sign, accept uint32) (vt *DefaultVotingThresholdPolicy, err error) {
	if init <= 0 || sign <= 0 || accept <= 0 {
		err = sebakerror.ErrorInvalidVotingThresholdPolicy
		return
	}
	if init > 100 || sign > 100 || accept > 100 {
		err = sebakerror.ErrorInvalidVotingThresholdPolicy
		return
	}

	vt = &DefaultVotingThresholdPolicy{
		init:       init,
		sign:       sign,
		accept:     accept,
		validators: 0,
	}

	return
}
