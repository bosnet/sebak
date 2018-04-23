package sebak

import "encoding/json"

type Ballot struct {
	Hash   string `json:"hash"`
	Vote   bool   `json:"vote"`
	Reason string `json:"reason"`
}

func (ballot Ballot) Serialize() (encoded []byte, err error) {
	encoded, err = json.Marshal(ballot)
	return
}

func (ballot Ballot) String() string {
	encoded, _ := json.MarshalIndent(ballot, "", "  ")
	return string(encoded)
}

func NewBallotFromJSON(b []byte) (ballot Ballot, err error) {
	if err = json.Unmarshal(b, &ballot); err != nil {
		return
	}

	return
}
