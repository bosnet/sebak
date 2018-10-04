package operation

import (
	"encoding/json"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/error"
)

type CongressVoting struct {
	Contract []byte
	Voting   struct {
		Start uint64
		End   uint64
	}
}

func (o CongressVoting) Serialize() (encoded []byte, err error) {
	encoded, err = json.Marshal(o)
	return
}
func (o CongressVoting) IsWellFormed([]byte, common.Config) (err error) {
	if len(o.Contract) == 0 {
		return errors.ErrorOperationBodyInsufficient
	}

	if o.Voting.End < o.Voting.Start {
		return errors.ErrorInvalidOperation
	}
	return
}
