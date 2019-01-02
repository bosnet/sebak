package ballot

import (
	"time"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/errors"
)

func CheckHasCorrectTime(timeStr string) error {
	var proposerConfirmed time.Time
	var err error
	if proposerConfirmed, err = common.ParseISO8601(timeStr); err != nil {
		return err
	}

	now := time.Now()
	timeStart := now.Add(time.Duration(-1) * common.BallotConfirmedTimeAllowDuration)
	timeEnd := now.Add(common.BallotConfirmedTimeAllowDuration)

	if proposerConfirmed.Before(timeStart) || proposerConfirmed.After(timeEnd) {
		return errors.MessageHasIncorrectTime
	}

	return nil
}
