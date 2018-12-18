package runner

import (
	"net/http"
)

const GetBallotPattern = "/ballots"

func (nh NetworkHandlerNode) GetBallotHandler(w http.ResponseWriter, r *http.Request) {
	rrs := nh.consensus.RunningRounds
	if len(rrs) < 1 {
		return
	}

	for _, rr := range rrs {
		for _, b := range rr.Ballots {
			nh.renderNodeItem(w, NodeItemBallot, b)
		}
	}

	return
}
