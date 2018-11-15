package operation

import (
	"encoding/json"
	"net/url"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/errors"
)

type CongressVotingResult struct {
	BallotStamps struct {
		Hash string   `json:"hash"`
		Urls []string `json:"urls"`
	} `json:"ballot_stamps"`
	Voters struct {
		Hash string   `json:"hash"`
		Urls []string `json:"urls"`
	} `json:"voters"`
	Result struct {
		Count uint64 `json:"count"`
		Yes   uint64 `json:"yes"`
		No    uint64 `json:"no"`
		ABS   uint64 `json:"abs"`
	} `json:"result"`
}

func NewCongressVotingResult(
	ballotHash string, ballotUrls []string,
	votersHash string, votersUrls []string,
	resultCount, resultYes, resultNo, resultABS uint64) CongressVotingResult {

	return CongressVotingResult{
		BallotStamps: struct {
			Hash string   `json:"hash"`
			Urls []string `json:"urls"`
		}{ballotHash, ballotUrls},
		Voters: struct {
			Hash string   `json:"hash"`
			Urls []string `json:"urls"`
		}{votersHash, votersUrls},
		Result: struct {
			Count uint64 `json:"count"`
			Yes   uint64 `json:"yes"`
			No    uint64 `json:"no"`
			ABS   uint64 `json:"abs"`
		}{resultCount, resultYes, resultNo, resultABS},
	}
}

func (o CongressVotingResult) Serialize() (encoded []byte, err error) {
	return json.Marshal(o)
}
func (o CongressVotingResult) IsWellFormed(common.Config) (err error) {
	if len(o.BallotStamps.Hash) == 0 {
		return errors.OperationBodyInsufficient
	}

	for _, u := range o.BallotStamps.Urls {
		if _, err := url.Parse(u); err != nil {
			return errors.InvalidOperation
		}
	}

	if len(o.Voters.Hash) == 0 {
		return errors.OperationBodyInsufficient
	}

	for _, u := range o.Voters.Urls {
		if _, err := url.Parse(u); err != nil {
			return errors.InvalidOperation
		}
	}

	if o.Result.Count != o.Result.Yes+o.Result.No+o.Result.ABS {
		return errors.InvalidOperation
	}

	return
}

func (o CongressVotingResult) HasFee(isSourceLinked bool) bool {
	return true
}
