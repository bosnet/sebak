package consensus

import (
	"encoding/json"
	"math"
	"sync"

	"github.com/spikeekips/sebak/lib/error"
	"github.com/spikeekips/sebak/lib/util"
)

type Voting string

const (
	VotingNOTYET Voting = "NOT-YET"
	VotingYES    Voting = "YES"
	VotingNO     Voting = "NO"
)

type VotingResultBallot struct {
	Hash   string // `Ballot.GetHash()`
	State  BallotState
	Voting Voting
	Reason string
}

func NewVotingResultBallotFromBallot(ballot Ballot) VotingResultBallot {
	return VotingResultBallot{
		Hash:   ballot.GetHash(),
		State:  ballot.B.State,
		Voting: ballot.B.Voting,
		Reason: ballot.B.Reason,
	}
}

type VotingResultBallots map[ /* NodeKey */ string]VotingResultBallot

type VotingResult struct {
	sync.Mutex

	ID          string      // ID is unique and sequenital
	MessageHash string      // MessageHash is `Message.Hash`
	State       BallotState // Latest `BallotState`
	Ballots     map[BallotState]VotingResultBallots
}

func NewVotingResult(ballot Ballot) (vr *VotingResult, err error) {
	ballots := map[BallotState]VotingResultBallots{
		BallotStateNONE:       VotingResultBallots{},
		BallotStateINIT:       VotingResultBallots{},
		BallotStateSIGN:       VotingResultBallots{},
		BallotStateACCEPT:     VotingResultBallots{},
		BallotStateALLCONFIRM: VotingResultBallots{},
	}

	ballots[ballot.B.State][ballot.B.NodeKey] = NewVotingResultBallotFromBallot(ballot)

	vr = &VotingResult{
		ID:          util.GetUniqueIDFromUUID(),
		MessageHash: ballot.GetMessage().GetHash(),
		State:       BallotStateNONE,
		Ballots:     ballots,
	}

	return
}

func (vr *VotingResult) SetState(state BallotState) bool {
	if vr.State >= state {
		return false
	}

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
	ballots, ok := vr.Ballots[ballot.GetState()]
	if !ok {
		return false
	}
	if _, ok := ballots[ballot.B.NodeKey]; !ok {
		return false
	}

	return true
}

func (vr *VotingResult) GetVotedBallotsByState(state BallotState) VotingResultBallots {
	return vr.Ballots[state]
}

func (vr *VotingResult) GetVotedCount(state BallotState) int {
	return len(vr.GetVotedBallotsByState(state))
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
	vr.Ballots[ballot.GetState()][ballot.B.NodeKey] = NewVotingResultBallotFromBallot(ballot)

	return
}

func (vr *VotingResult) CanCheckThreshold(state BallotState, threshold uint32) bool {
	if threshold < 1 {
		return false
	}
	if state == BallotStateNONE {
		return false
	}
	if vr.GetVotedCount(state) < int(threshold) {
		return false
	}

	return true
}

func (vr *VotingResult) CheckThreshold(state BallotState, threshold uint32) bool {
	if threshold < 1 {
		return false
	}
	if state == BallotStateNONE {
		return false
	}
	if vr.GetVotedCount(state) < int(threshold) {
		return false
	}

	var yes int
	for _, vrb := range vr.GetVotedBallotsByState(state) {
		if vrb.Voting == VotingYES {
			yes += 1
		}
	}
	return yes >= int(threshold)
}

var CheckVotingThresholdSequence = []BallotState{
	BallotStateACCEPT,
	BallotStateSIGN,
	BallotStateINIT,
}

func (vr *VotingResult) GetResult(policy VotingThresholdPolicy) (BallotState, bool) {
	if vr.State == BallotStateALLCONFIRM {
		return BallotStateALLCONFIRM, false
	}

	for _, state := range CheckVotingThresholdSequence {
		if state < vr.State {
			break
		}

		t := policy.GetThreshold(state)
		if t < 1 {
			continue
		}
		if vr.CheckThreshold(state, t) {
			return state, true
		}
	}

	return vr.State, false
}

func (vr *VotingResult) CanGetResult(policy VotingThresholdPolicy) bool {
	if vr.State == BallotStateALLCONFIRM {
		return false
	}

	for _, state := range CheckVotingThresholdSequence {
		t := policy.GetThreshold(state)
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
	GetThreshold(BallotState) uint32
	SetValidators(uint64) error
}

type DefaultVotingThresholdPolicy struct {
	init   uint32 // must be percentile
	sign   uint32
	accept uint32

	validators uint64
}

func (vt *DefaultVotingThresholdPolicy) GetValidators() uint64 {
	return vt.validators
}

func (vt *DefaultVotingThresholdPolicy) SetValidators(v uint64) error {
	if v < 1 {
		return sebak_error.ErrorVotingThresholdInvalidValidators
	}

	vt.validators = v

	return nil
}

func (vt *DefaultVotingThresholdPolicy) GetThreshold(state BallotState) uint32 {
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
		err = sebak_error.ErrorInvalidVotingThresholdPolicy
		return
	}
	if init > 100 || sign > 100 || accept > 100 {
		err = sebak_error.ErrorInvalidVotingThresholdPolicy
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
