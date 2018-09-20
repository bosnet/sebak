package transaction

import (
	"encoding/json"
)

type Voting struct {
	Start        string
	End          string
}

type OperationBodyCongressVoting struct {
	Contract     string
	Voting       Voting
}

func (o OperationBodyCongressVoting) Serialize() (encoded []byte, err error) {
	encoded, err = json.Marshal(o)
	return
}

func (o OperationBodyCongressVoting) IsWellFormed([]byte) (err error) {
	return
}

