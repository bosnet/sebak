package sebakcommon

type BallotState uint

const (
	BallotStateNONE BallotState = iota
	BallotStateINIT
	BallotStateSIGN
	BallotStateACCEPT
	BallotStateALLCONFIRM
)

func (s BallotState) String() string {
	switch s {
	case BallotStateINIT:
		return "INIT"
	case BallotStateSIGN:
		return "SIGN"
	case BallotStateACCEPT:
		return "ACCEPT"
	case BallotStateALLCONFIRM:
		return "ALLCONFIRM"
	default:
		return ""
	}
}

func (s BallotState) IsValid() bool {
	switch s {
	case BallotStateINIT:
	case BallotStateSIGN:
	case BallotStateACCEPT:
	default:
		return false
	}

	return true
}

func (s BallotState) Next() BallotState {
	switch s {
	case BallotStateINIT:
		return BallotStateSIGN
	case BallotStateSIGN:
		return BallotStateACCEPT
	case BallotStateACCEPT:
		return BallotStateALLCONFIRM
	default:
		return BallotStateNONE
	}
}

func (s BallotState) IsValidForVote() bool {
	switch s {
	case BallotStateSIGN:
	case BallotStateACCEPT:
	default:
		return false
	}

	return true
}
