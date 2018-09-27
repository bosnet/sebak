package transaction

import (
	"boscoin.io/sebak/lib/error"
	"encoding/json"
	"time"
)

type OperationBodyCongressVoting struct {
	Contract string
	Voting   struct {
		Start string
		End   string
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
	if _, err := time.Parse(time.RFC3339Nano, o.Voting.Start); err != nil {
		return errors.ErrorInvalidOperation
	}

	if _, err := time.Parse(time.RFC3339Nano, o.Voting.End); err != nil {
		return errors.ErrorInvalidOperation
	}
	return
}
