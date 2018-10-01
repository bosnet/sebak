package consensus

import (
	"encoding/json"
	"math"
	"sync"

	"boscoin.io/sebak/lib/ballot"
	"boscoin.io/sebak/lib/error"
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

func (vt *ISAACVotingThresholdPolicy) SetValidators(n int) error {
	if n < 1 {
		return errors.ErrorVotingThresholdInvalidValidators
	}

	vt.validators = n

	return nil
}

func (vt *ISAACVotingThresholdPolicy) Connected() int {
	vt.RLock()
	defer vt.RUnlock()

	return vt.connected
}

func (vt *ISAACVotingThresholdPolicy) SetConnected(n int) error {
	if n < 1 {
		return errors.ErrorVotingThresholdInvalidValidators
	}

	vt.Lock()
	defer vt.Unlock()

	vt.connected = n

	return nil
}

func (vt *ISAACVotingThresholdPolicy) Threshold(state ballot.State) int {
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
		err = errors.ErrorInvalidVotingThresholdPolicy
		return
	}
	if threshold > 100 {
		err = errors.ErrorInvalidVotingThresholdPolicy
		return
	}

	vt = &ISAACVotingThresholdPolicy{
		threshold:  threshold,
		validators: 0,
	}

	return
}
