package consensus

import (
	"encoding/json"
	"math"
	"sync"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/error"
)

type ISAACVotingThresholdPolicy struct {
	sync.RWMutex

	sign   int
	accept int

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

func (vt *ISAACVotingThresholdPolicy) Threshold(state common.BallotState) int {
	var t int
	switch state {
	case common.BallotStateSIGN:
		t = vt.sign
	case common.BallotStateACCEPT:
		t = vt.accept
	}

	v := float64(vt.validators) * (float64(t) / float64(100))
	threshold := int(math.Ceil(v))

	// in SIGN state, proposer assumes to say VotingYES
	if state == common.BallotStateSIGN {
		threshold = threshold - 1
	}

	if threshold > 0 {
		return threshold
	}

	return 0
}

func (vt *ISAACVotingThresholdPolicy) MarshalJSON() ([]byte, error) {
	vt.RLock()
	defer vt.RUnlock()

	return json.Marshal(map[string]interface{}{
		"sign":       vt.sign,
		"accept":     vt.accept,
		"validators": vt.validators,
		"connected":  vt.connected,
	})
}

func NewDefaultVotingThresholdPolicy(sign, accept int) (vt *ISAACVotingThresholdPolicy, err error) {
	if sign <= 0 || accept <= 0 {
		err = errors.ErrorInvalidVotingThresholdPolicy
		return
	}
	if sign > 100 || accept > 100 {
		err = errors.ErrorInvalidVotingThresholdPolicy
		return
	}

	vt = &ISAACVotingThresholdPolicy{
		sign:       sign,
		accept:     accept,
		validators: 0,
	}

	return
}
