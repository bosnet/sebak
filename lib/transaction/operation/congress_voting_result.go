package operation

import (
	"encoding/json"

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
	CongressVotingHash string `json:"congress_voting_hash"`
	Membership         struct {
		Hash string   `json:"hash"`
		Urls []string `json:"urls"`
	} `json:"membership"`
}

func NewCongressVotingResult(
	ballotHash string, ballotUrls []string,
	votersHash string, votersUrls []string,
	membershipHash string, membershipUrls []string,
	resultCount, resultYes, resultNo, resultABS uint64,
	congressVotingHash string,
) CongressVotingResult {
	return CongressVotingResult{
		BallotStamps: struct {
			Hash string   `json:"hash"`
			Urls []string `json:"urls"`
		}{ballotHash, ballotUrls},
		Voters: struct {
			Hash string   `json:"hash"`
			Urls []string `json:"urls"`
		}{votersHash, votersUrls},
		Membership: struct {
			Hash string   `json:"hash"`
			Urls []string `json:"urls"`
		}{membershipHash, membershipUrls},
		Result: struct {
			Count uint64 `json:"count"`
			Yes   uint64 `json:"yes"`
			No    uint64 `json:"no"`
			ABS   uint64 `json:"abs"`
		}{resultCount, resultYes, resultNo, resultABS},
		CongressVotingHash: congressVotingHash,
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
		if _, err := common.StrictURLParse(u); err != nil {
			return errors.InvalidOperation.Clone().SetData("error", err)
		}
	}

	if len(o.Voters.Hash) == 0 {
		return errors.OperationBodyInsufficient
	}

	for _, u := range o.Voters.Urls {
		if _, err := common.StrictURLParse(u); err != nil {
			return errors.InvalidOperation.Clone().SetData("error", err)
		}
	}

	if len(o.Membership.Hash) == 0 {
		return errors.OperationBodyInsufficient
	}

	for _, u := range o.Membership.Urls {
		if _, err := common.StrictURLParse(u); err != nil {
			return errors.InvalidOperation.Clone().SetData("error", err)
		}
	}

	if o.Result.Count != o.Result.Yes+o.Result.No+o.Result.ABS {
		return errors.InvalidOperation
	}

	return
}

func (o CongressVotingResult) HasFee() bool {
	return true
}
