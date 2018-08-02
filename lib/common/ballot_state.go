package sebakcommon

import (
	"fmt"
)

type BallotState uint

const (
	BallotStateNONE BallotState = iota
	BallotStateTXSHARE
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
		return "TXSHARE"
	case 2:
		return "INIT"
	case 3:
		return "SIGN"
	case 4:
		return "ACCEPT"
	case 5:
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
	case "TXSHARE":
		c = 1
	case "INIT":
		c = 2
	case "SIGN":
		c = 3
	case "ACCEPT":
		c = 4
	case "ALL-CONFIRM":
		c = 5
	}

	*s = BallotState(c)

	return
}
