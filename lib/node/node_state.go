package node

import (
	"fmt"
)

type State uint

const (
	StateNONE State = iota
	StateBOOTING
	StateSYNC
	StateCONSENSUS
	StateTERMINATING
)

var NodeInitState = StateNONE

func (s State) String() string {
	switch s {
	case 0:
		return "NONE"
	case 1:
		return "BOOTING"
	case 2:
		return "SYNC"
	case 3:
		return "CONSENSUS"
	case 4:
		return "TERMINATING"
	}

	return ""
}

func (s State) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("\"%s\"", s.String())), nil
}

func (s *State) UnmarshalJSON(b []byte) (err error) {
	var c int
	switch string(b[1 : len(b)-1]) {
	case "NONE":
		c = 0
	case "BOOTING":
		c = 1
	case "SYNC":
		c = 2
	case "CONSENSUS":
		c = 3
	case "TERMINATING":
		c = 4
	}

	*s = State(c)

	return
}
