package operation

import (
	"encoding/json"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/errors"
)

type CongressVoting struct {
	Contract []byte `json:"contract"`
	Voting   struct {
		Start uint64 `json:"start"`
		End   uint64 `json:"end"`
	} `json:"voting"`
}

func NewCongressVoting(contract []byte, start, end uint64) CongressVoting {

	return CongressVoting{
		Contract: contract,
		Voting: struct {
			Start uint64 `json:"start"`
			End   uint64 `json:"end"`
		}{
			Start: start,
			End:   end,
		},
	}
}

func (o CongressVoting) Serialize() (encoded []byte, err error) {
	encoded, err = json.Marshal(o)
	return
}
func (o CongressVoting) IsWellFormed(common.Config) (err error) {
	if len(o.Contract) == 0 {
		return errors.OperationBodyInsufficient
	}

	if o.Voting.End < o.Voting.Start {
		return errors.InvalidOperation
	}
	return
}

func (o CongressVoting) HasFee() bool {
	return true
}
