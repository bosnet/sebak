package sebak

import (
	"encoding/json"
	"math"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/error"
)

type VotingHole string

const (
	VotingNOTYET VotingHole = "NOT-YET"
	VotingYES    VotingHole = "YES"
	VotingNO     VotingHole = "NO"
)

type VotingResultBallot struct {
	Hash       string // `Ballot.GetHash()`
	State      sebakcommon.BallotState
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
	State         sebakcommon.BallotState
	PreviousState sebakcommon.BallotState

	ID          string     // ID is unique and sequential
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
	if vs.IsEmpty() {
		return false
	}
	if vs.VotingHole == VotingNO {
		return true
	}
	if vs.State == sebakcommon.BallotStateALLCONFIRM {
		return true
	}

	return false
}

func (vs VotingStateStaging) IsStorable() bool {
	if !vs.IsClosed() {
		return false
	}
	if vs.State != sebakcommon.BallotStateALLCONFIRM {
		return false
	}
	if vs.VotingHole == VotingNO {
		return false
	}

	return true
}

type VotingResult struct {
	sebakcommon.SafeLock

	ID          string                  // ID is unique and sequential
	MessageHash string                  // MessageHash is `Message.Hash`
	Source      string                  // Source is `Ballot.Source()`
	State       sebakcommon.BallotState // Latest `BallotState`
	Ballots     map[sebakcommon.BallotState]VotingResultBallots
	Staging     []VotingStateStaging // state changing histories
}

func NewVotingResult(ballot Ballot) (vr *VotingResult, err error) {
	ballots := map[sebakcommon.BallotState]VotingResultBallots{
		sebakcommon.BallotStateNONE:       VotingResultBallots{},
		sebakcommon.BallotStateINIT:       VotingResultBallots{},
		sebakcommon.BallotStateSIGN:       VotingResultBallots{},
		sebakcommon.BallotStateACCEPT:     VotingResultBallots{},
		sebakcommon.BallotStateALLCONFIRM: VotingResultBallots{},
	}

	ballots[ballot.State()][ballot.B.NodeKey] = NewVotingResultBallotFromBallot(ballot)

	vr = &VotingResult{
		ID:          sebakcommon.GetUniqueIDFromUUID(),
		MessageHash: ballot.MessageHash(),
		Source:      ballot.Source(),
		State:       ballot.State(),
		Ballots:     ballots,
		Staging:     make([]VotingStateStaging, 0),
	}

	return
}

func (vr *VotingResult) IsClosed() bool {
	vr.Lock()
	defer vr.Unlock()
	return vr.LatestStaging().IsClosed()
}

func (vr *VotingResult) SetState(state sebakcommon.BallotState) bool {
	if vr.State >= state {
		return false
	}

	vr.Lock()
	defer vr.Unlock()

	vr.State = state

	return true
}

func (vr *VotingResult) Serialize() (encoded []byte, err error) {
	vr.Lock()
	defer vr.Unlock()
	encoded, err = json.Marshal(vr)
	return
}

func (vr *VotingResult) String() string {
	vr.Lock()
	defer vr.Unlock()
	encoded, _ := json.MarshalIndent(vr, "", "  ")
	return string(encoded)
}

func (vr *VotingResult) IsVoted(ballot Ballot) bool {
	vr.Lock()
	defer vr.Unlock()
	ballots, ok := vr.Ballots[ballot.State()]
	if !ok {
		return false
	}
	if _, ok := ballots[ballot.B.NodeKey]; !ok {
		return false
	}

	return true
}

func (vr *VotingResult) VotedBallotsByState(state sebakcommon.BallotState) VotingResultBallots {
	vr.Lock()
	defer vr.Unlock()
	return vr.Ballots[state]
}

func (vr *VotingResult) VotedCount(state sebakcommon.BallotState) int {
	vr.Lock()
	defer vr.Unlock()
	return len(vr.VotedBallotsByState(state))
}

var VotingResultCheckerFuns = []sebakcommon.CheckerFunc{
	checkBallotResultValidHash,
}

func (vr *VotingResult) Add(ballot Ballot) (err error) {
	vr.Lock()
	defer vr.Unlock()

	checker := &VotingResultChecker{
		DefaultChecker: sebakcommon.DefaultChecker{VotingResultCheckerFuns},
		VotingResult:   vr,
		Ballot:         ballot,
	}
	if err = sebakcommon.RunChecker(checker, sebakcommon.DefaultDeferFunc); err != nil {
		return
	}
	vr.Ballots[ballot.State()][ballot.B.NodeKey] = NewVotingResultBallotFromBallot(ballot)

	return
}

func (vr *VotingResult) CanCheckThreshold(state sebakcommon.BallotState, threshold int) bool {
	vr.Lock()
	defer vr.Unlock()
	if threshold < 1 {
		return false
	}
	if state == sebakcommon.BallotStateNONE {
		return false
	}
	if vr.VotedCount(state) < int(threshold) {
		return false
	}

	return true
}

func (vr *VotingResult) CheckThreshold(state sebakcommon.BallotState, policy sebakcommon.VotingThresholdPolicy) (VotingHole, bool) {
	vr.Lock()
	defer vr.Unlock()
	threshold := policy.Threshold(state)
	if threshold < 1 {
		return VotingNOTYET, false
	}
	if state == sebakcommon.BallotStateNONE {
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

	log.Debug(
		"check threshold",
		"state", state,
		"threshold", threshold,
		"yes", yes,
		"no", no,
		"policy", policy,
	)

	if yes >= threshold {
		return VotingYES, true
	} else if no >= threshold {
		return VotingNO, true
	}

	// check draw!
	total := policy.Validators()
	voted := yes + no
	if total-voted < threshold-yes && total-voted < threshold-no { // draw
		return VotingNO, true
	}

	return VotingNOTYET, false
}

var CheckVotingThresholdSequence = []sebakcommon.BallotState{
	sebakcommon.BallotStateACCEPT,
	sebakcommon.BallotStateSIGN,
	sebakcommon.BallotStateINIT,
}

func (vr *VotingResult) MakeResult(policy sebakcommon.VotingThresholdPolicy) (VotingHole, sebakcommon.BallotState, bool) {
	vr.Lock()
	defer vr.Unlock()
	if vr.State == sebakcommon.BallotStateALLCONFIRM {
		return VotingNOTYET, sebakcommon.BallotStateALLCONFIRM, false
	}

	for _, state := range CheckVotingThresholdSequence {
		if state < vr.State {
			break
		}

		t := policy.Threshold(state)
		if t < 1 {
			continue
		}
		votingHole, ended := vr.CheckThreshold(state, policy)
		if ended {
			return votingHole, state, true
		}
	}

	return VotingNOTYET, vr.State, false
}

func (vr *VotingResult) ChangeState(votingHole VotingHole, state sebakcommon.BallotState) (vs VotingStateStaging, err error) {
	if votingHole == VotingYES && !vr.SetState(state.Next()) {
		err = sebakerror.ErrorVotingResultFailedToSetState
		return
	}

	vs = vr.MakeStaging(votingHole, state, vr.State, state)

	vr.Lock()
	defer vr.Unlock()

	vr.Staging = append(vr.Staging, vs)

	return
}

func (vr *VotingResult) MakeStaging(votingHole VotingHole, previousState, nextState, votingState sebakcommon.BallotState) VotingStateStaging {
	vr.Lock()
	defer vr.Unlock()
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
	vr.Lock()
	defer vr.Unlock()
	if len(vr.Staging) < 1 {
		return VotingStateStaging{}
	}
	return vr.Staging[len(vr.Staging)-1]
}

func (vr *VotingResult) CanGetResult(policy sebakcommon.VotingThresholdPolicy) bool {
	vr.Lock()
	defer vr.Unlock()
	if vr.State == sebakcommon.BallotStateALLCONFIRM {
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

type ISAACVotingThresholdPolicy struct {
	init   int // must be percentile
	sign   int
	accept int

	validators int
	connected  int
}

func (vt *ISAACVotingThresholdPolicy) String() string {
	o := sebakcommon.MustJSONMarshal(map[string]interface{}{
		"init":       vt.init,
		"sign":       vt.sign,
		"accept":     vt.accept,
		"validators": vt.validators,
	})

	return string(o)
}

func (vt *ISAACVotingThresholdPolicy) Validators() int {
	return vt.validators
}

func (vt *ISAACVotingThresholdPolicy) SetValidators(n int) error {
	if n < 1 {
		return sebakerror.ErrorVotingThresholdInvalidValidators
	}

	vt.validators = n

	return nil
}

func (vt *ISAACVotingThresholdPolicy) Connected() int {
	return vt.connected
}

func (vt *ISAACVotingThresholdPolicy) SetConnected(n int) error {
	if n < 1 {
		return sebakerror.ErrorVotingThresholdInvalidValidators
	}

	vt.connected = n

	return nil
}

func (vt *ISAACVotingThresholdPolicy) Threshold(state sebakcommon.BallotState) int {
	var t int
	var va int
	switch state {
	case sebakcommon.BallotStateINIT:
		t = vt.init
		va = vt.connected + 1
	case sebakcommon.BallotStateSIGN:
		t = vt.sign
		va = vt.validators
	case sebakcommon.BallotStateACCEPT:
		t = vt.accept
		va = vt.validators
	}

	v := float64(va) * (float64(t) / float64(100))
	return int(math.Ceil(v))
}

func (vt *ISAACVotingThresholdPolicy) Reset(state sebakcommon.BallotState, threshold int) (err error) {
	if threshold <= 0 {
		err = sebakerror.ErrorInvalidVotingThresholdPolicy
		return
	}

	if threshold > 100 {
		err = sebakerror.ErrorInvalidVotingThresholdPolicy
		return
	}

	switch state {
	case sebakcommon.BallotStateINIT:
		vt.init = threshold
	case sebakcommon.BallotStateSIGN:
		vt.sign = threshold
	case sebakcommon.BallotStateACCEPT:
		vt.accept = threshold
	}

	return nil
}

func NewDefaultVotingThresholdPolicy(init, sign, accept int) (vt *ISAACVotingThresholdPolicy, err error) {
	if init <= 0 || sign <= 0 || accept <= 0 {
		err = sebakerror.ErrorInvalidVotingThresholdPolicy
		return
	}
	if init > 100 || sign > 100 || accept > 100 {
		err = sebakerror.ErrorInvalidVotingThresholdPolicy
		return
	}

	vt = &ISAACVotingThresholdPolicy{
		init:       init,
		sign:       sign,
		accept:     accept,
		validators: 0,
	}

	return
}
