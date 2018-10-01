package transaction

import (
	"boscoin.io/sebak/lib/error"
	"encoding/json"
)

type OperationBodyCongressVoting struct {
	Contract []byte
	Voting   struct {
		Start uint64
		End   uint64
	}
}

func (o OperationBodyCongressVoting) Serialize() (encoded []byte, err error) {
	encoded, err = json.Marshal(o)
	return
}
func (o OperationBodyCongressVoting) IsWellFormed([]byte) (err error) {
	if len(o.Contract) == 0 {
		return errors.ErrorOperationBodyInsufficient
	}

	if o.Voting.End < o.Voting.Start {
		return errors.ErrorInvalidOperation
	}
	return
}
