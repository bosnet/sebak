package sebak

import (
	"math"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/error"
)

type ISAACVotingThresholdPolicy struct {
	threshold int

	validators int
	connected  int
}

func (vt *ISAACVotingThresholdPolicy) String() string {
	o := sebakcommon.MustJSONMarshal(map[string]interface{}{
		"threshold":  vt.threshold,
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

func (vt *ISAACVotingThresholdPolicy) Threshold() int {
	v := float64(vt.validators) * (float64(vt.threshold) / float64(100))
	return int(math.Ceil(v))
}

func (vt *ISAACVotingThresholdPolicy) Reset(threshold int) (err error) {
	if threshold <= 0 {
		err = sebakerror.ErrorInvalidVotingThresholdPolicy
		return
	}

	if threshold > 100 {
		err = sebakerror.ErrorInvalidVotingThresholdPolicy
		return
	}

	vt.threshold = threshold

	return nil
}

func NewDefaultVotingThresholdPolicy(threshold int) (vt *ISAACVotingThresholdPolicy, err error) {
	if threshold <= 0 {
		err = sebakerror.ErrorInvalidVotingThresholdPolicy
		return
	}
	if threshold > 100 {
		err = sebakerror.ErrorInvalidVotingThresholdPolicy
		return
	}

	vt = &ISAACVotingThresholdPolicy{
		threshold:  threshold,
		validators: 0,
	}

	return
}
