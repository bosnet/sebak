package runner

import (
	"net/http"

	api "boscoin.io/sebak/lib/node/runner/node_api"
)

const GetBallotPattern = "/ballots"

func (nh NetworkHandlerNode) GetBallotHandler(w http.ResponseWriter, r *http.Request) {
	rrs := nh.consensus.RunningRounds
	if len(rrs) < 1 {
		return
	}

	for _, rr := range rrs {
		for _, b := range rr.Ballots {
			nh.renderNodeItem(w, api.NodeItemBallot, b)
		}
	}

	return
}
