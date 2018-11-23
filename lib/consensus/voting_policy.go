package consensus

import (
	"encoding/json"
	"math"
	"sync"

	"boscoin.io/sebak/lib/errors"
)

type ISAACVotingThresholdPolicy struct {
	sync.RWMutex

	threshold  int
	validators int
	connected  int
}

func (vt *ISAACVotingThresholdPolicy) Validators() int {
	return vt.validators
}

func (vt *ISAACVotingThresholdPolicy) SetValidators(n int) {
	if n < 1 {
		panic(errors.VotingThresholdInvalidValidators)
	}
	vt.validators = n
}

func (vt *ISAACVotingThresholdPolicy) Connected() int {
	vt.RLock()
	defer vt.RUnlock()

	return vt.connected
}

func (vt *ISAACVotingThresholdPolicy) SetConnected(n int) {
	if n < 1 {
		panic(errors.VotingThresholdInvalidValidators)
	}

	vt.Lock()
	defer vt.Unlock()

	vt.connected = n
}

func (vt *ISAACVotingThresholdPolicy) Threshold() int {
	v := float64(vt.validators) * (float64(vt.threshold) / float64(100))
	threshold := int(math.Ceil(v))

	if threshold < 0 {
		return 0
	}

	return threshold
}

func (vt *ISAACVotingThresholdPolicy) MarshalJSON() ([]byte, error) {
	vt.RLock()
	defer vt.RUnlock()

	return json.Marshal(map[string]interface{}{
		"threshold":  vt.threshold,
		"validators": vt.validators,
		"connected":  vt.connected,
	})
}

func NewDefaultVotingThresholdPolicy(threshold int) (vt *ISAACVotingThresholdPolicy, err error) {
	if threshold <= 0 {
		err = errors.InvalidVotingThresholdPolicy
		return
	}
	if threshold > 100 {
		err = errors.InvalidVotingThresholdPolicy
		return
	}

	vt = &ISAACVotingThresholdPolicy{
		threshold:  threshold,
		validators: 0,
	}

	return
}
