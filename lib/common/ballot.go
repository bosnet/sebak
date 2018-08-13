package sebakcommon

type BallotState string

const (
	BallotStateINIT   BallotState = "INIT"
	BallotStateSIGN   BallotState = "SIGN"
	BallotStateACCEPT BallotState = "ACCEPT"
)

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

func (s BallotState) IsValidForVote() bool {
	switch s {
	case BallotStateSIGN:
	case BallotStateACCEPT:
	default:
		return false
	}

	return true
}
