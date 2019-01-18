package ballot

import (
	"time"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/errors"
)

func CheckHasCorrectTime(timeStr string) error {
	var t time.Time
	var err error
	if t, err = common.ParseISO8601(timeStr); err != nil {
		return err
	}
	now := time.Now()
	timeStart := now.Add(time.Duration(-1) * common.BallotConfirmedTimeAllowDuration)
	timeEnd := now.Add(common.BallotConfirmedTimeAllowDuration)
	if t.Before(timeStart) || t.After(timeEnd) {
		return errors.MessageHasIncorrectTime
	}

	return nil
}
