package sebak

import (
	"math"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/error"
)

type ISAACVotingThresholdPolicy struct {
	sign   int
	accept int

	validators int
	connected  int
}

func (vt *ISAACVotingThresholdPolicy) String() string {
	o := sebakcommon.MustJSONMarshal(map[string]interface{}{
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
	switch state {
	case sebakcommon.BallotStateSIGN:
		t = vt.sign
	case sebakcommon.BallotStateACCEPT:
		t = vt.accept
	}

	v := float64(vt.validators) * (float64(t) / float64(100))
	threshold := int(math.Ceil(v))

	// in SIGN state, proposer assumes to say VotingYES
	if state == sebakcommon.BallotStateSIGN {
		threshold = threshold - 1
	}

	if threshold > 0 {
		return threshold
	}

	return 0
}

func NewDefaultVotingThresholdPolicy(sign, accept int) (vt *ISAACVotingThresholdPolicy, err error) {
	if sign <= 0 || accept <= 0 {
		err = sebakerror.ErrorInvalidVotingThresholdPolicy
		return
	}
	if sign > 100 || accept > 100 {
		err = sebakerror.ErrorInvalidVotingThresholdPolicy
		return
	}

	vt = &ISAACVotingThresholdPolicy{
		sign:       sign,
		accept:     accept,
		validators: 0,
	}

	return
}
