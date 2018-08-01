package sebak

import (
	"math"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/error"
)

type ISAACVotingThresholdPolicy struct {
	init   int // [TODO] Change to Percent type
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
