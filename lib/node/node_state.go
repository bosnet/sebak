package sebaknode

import (
	"fmt"
)

type NodeState uint

const (
	NodeStateNONE NodeState = iota
	NodeStateBOOTING
	NodeStateSYNC
	NodeStateCONSENSUS
	NodeStateTERMINATING
)

var NodeInitState = NodeStateNONE

func (s NodeState) String() string {
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

func (s NodeState) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("\"%s\"", s.String())), nil
}

func (s *NodeState) UnmarshalJSON(b []byte) (err error) {
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

	*s = NodeState(c)

	return
}
