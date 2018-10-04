package operation

import (
	"encoding/json"
	"net/url"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/error"
)

type CongressVotingResult struct {
	BallotStamps struct {
		Hash string
		Urls []string
	}
	Voters struct {
		Hash string
		Urls []string
	}
	Result struct {
		Count uint64
		Yes   uint64
		No    uint64
		ABS   uint64
	}
}

func (o CongressVotingResult) Serialize() (encoded []byte, err error) {
	return json.Marshal(o)
}
func (o CongressVotingResult) IsWellFormed([]byte, common.Config) (err error) {
	if len(o.BallotStamps.Hash) == 0 {
		return errors.ErrorOperationBodyInsufficient
	}

	for _, u := range o.BallotStamps.Urls {
		if _, err := url.Parse(u); err != nil {
			return errors.ErrorInvalidOperation
		}
	}

	if len(o.Voters.Hash) == 0 {
		return errors.ErrorOperationBodyInsufficient
	}

	for _, u := range o.Voters.Urls {
		if _, err := url.Parse(u); err != nil {
			return errors.ErrorInvalidOperation
		}
	}

	if o.Result.Count != o.Result.Yes+o.Result.No+o.Result.ABS {
		return errors.ErrorInvalidOperation
	}

	return
}
