package node

import (
	"fmt"
)

type State uint

const (
	StateBOOTING State = iota
	StateCONSENSUS
	StateSYNC
	StateWATCH
)

func (s State) String() string {
	switch s {
	case 0:
		return "BOOTING"
	case 1:
		return "CONSENSUS"
	case 2:
		return "SYNC"
	case 3:
		return "WATCH"
	}

	return ""
}

func (s State) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("\"%s\"", s.String())), nil
}

func (s *State) UnmarshalJSON(b []byte) (err error) {
	var c int
	switch string(b[1 : len(b)-1]) {
	case "BOOTING":
		c = 0
	case "CONSENSUS":
		c = 1
	case "SYNC":
		c = 2
	case "WATCH":
		c = 3
	}

	*s = State(c)

	return
}
