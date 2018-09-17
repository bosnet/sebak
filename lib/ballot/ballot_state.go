package ballot

type State uint

const (
	StateNONE State = iota
	StateINIT
	StateSIGN
	StateACCEPT
	StateALLCONFIRM
)

func (s State) String() string {
	switch s {
	case StateINIT:
		return "INIT"
	case StateSIGN:
		return "SIGN"
	case StateACCEPT:
		return "ACCEPT"
	case StateALLCONFIRM:
		return "ALLCONFIRM"
	default:
		return ""
	}
}

func (s State) IsValid() bool {
	switch s {
	case StateINIT:
	case StateSIGN:
	case StateACCEPT:
	default:
		return false
	}

	return true
}

func (s State) Next() State {
	switch s {
	case StateINIT:
		return StateSIGN
	case StateSIGN:
		return StateACCEPT
	case StateACCEPT:
		return StateALLCONFIRM
	default:
		return StateNONE
	}
}

func (s State) IsValidForVote() bool {
	switch s {
	case StateSIGN:
	case StateACCEPT:
	default:
		return false
	}

	return true
}
