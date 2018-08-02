package sebakcommon

import (
	"fmt"
)

type BallotState uint

const (
	BallotStateNONE BallotState = iota
	BallotStateINIT
	BallotStateSIGN
	BallotStateACCEPT
	BallotStateALLCONFIRM
)

var BallotInitState = BallotStateNONE

func (s BallotState) String() string {
	switch s {
	case 0:
		return "NONE"
	case 1:
		return "INIT"
	case 2:
		return "SIGN"
	case 3:
		return "ACCEPT"
	case 4:
		return "ALL-CONFIRM"
	}

	return ""
}

func (s BallotState) Next() BallotState {
	n := s + 1
	if n > BallotStateALLCONFIRM {
		return BallotStateNONE
	}

	return n
}

func (s BallotState) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("\"%s\"", s.String())), nil
}

func (s *BallotState) UnmarshalJSON(b []byte) (err error) {
	var c int
	switch string(b[1 : len(b)-1]) {
	case "NONE":
		c = 0
	case "INIT":
		c = 1
	case "SIGN":
		c = 2
	case "ACCEPT":
		c = 3
	case "ALL-CONFIRM":
		c = 4
	}

	*s = BallotState(c)

	return
}
