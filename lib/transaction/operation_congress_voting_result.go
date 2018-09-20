package transaction

import (
	"encoding/json"
)

type BallotStamps struct {
	Hash   string
	urls   []string
}

type Voters struct {
	Hash   string
	urls   []string
}

type Result struct {
	Count  uint64
	Yes    uint64
	No     uint64
	ABS    uint64
}

type OperationBodyCongressVotingResult struct {
	BallotStamps BallotStamps
	Voters       Voters
	Result       Result
}

func (o OperationBodyCongressVotingResult) Serialize() (encoded []byte, err error) {
	encoded, err = json.Marshal(o)
	return
}

func (o OperationBodyCongressVotingResult) IsWellFormed([]byte) (err error) {

	return
}
