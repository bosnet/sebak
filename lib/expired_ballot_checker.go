package sebak

import (
	"boscoin.io/sebak/lib/common"
	"time"
)

type ExpiredBallotChecker struct {
	interval time.Duration

	WaitingResult  map[ /*VotingResultHash*/ string] /*Message.GetHash() */ string
	VotingResult   map[ /*VotingResultHash*/ string] /*Message.GetHash() */ string
	ReservedResult map[ /*VotingResultHash*/ string] /*Message.GetHash() */ string
}

func NewExpiredBallotChecker(curr *BallotBoxes, interval time.Duration) *ExpiredBallotChecker {
	e := &ExpiredBallotChecker{}

	e.WaitingResult = extractResultsFrom(curr.Results, curr.WaitingBox.Hashes)
	e.VotingResult = extractResultsFrom(curr.Results, curr.VotingBox.Hashes)
	e.ReservedResult = extractResultsFrom(curr.Results, curr.ReservedBox.Hashes)
	e.interval = interval

	return e
}

func extractResultsFrom(from map[string]*VotingResult, Hashes map[string]bool) (ret map[string]string) {
	ret = make(map[ /*VotingResultHash*/ string]string)
	for hash, _ := range Hashes {
		if vr, ok := from[hash]; ok {
			byteArr, err := sebakcommon.MakeObjectHash(vr)
			log.Error("failed to make voting result hash", "VotingResult", vr, "error", err)
			ret[string(byteArr)] = hash
		}
	}
	return
}

//func (e *ExpiredBallotChecker) Run() {
//	ticker := time.NewTicker(e.interval)
//	go func() {
//		for _ = range ticker.C {
//			e.TakeSnapshot(currentBallotBoxes)
//		}
//	}()
//
//}

func (e *ExpiredBallotChecker) TakeSnapshot(curr *BallotBoxes) ([]string, []string, []string) {
	currWaitingResult := extractResultsFrom(curr.Results, curr.WaitingBox.Hashes)
	currVotingResult := extractResultsFrom(curr.Results, curr.VotingBox.Hashes)
	currReservedResult := extractResultsFrom(curr.Results, curr.ReservedBox.Hashes)

	goToReservedFromWaiting := findRemainder(&e.WaitingResult, &currWaitingResult)
	goToReservedFromVoting := findRemainder(&e.VotingResult, &currVotingResult)

	goOut := findRemainder(&e.ReservedResult, &currReservedResult)

	e.WaitingResult = currWaitingResult
	e.VotingResult = currVotingResult
	e.ReservedResult = currReservedResult

	return goToReservedFromWaiting, goToReservedFromVoting, goOut
}

func findRemainder(prev *map[string]string, curr *map[string]string) (ret []string) {
	ret = []string{}
	for key := range *prev {
		if messageHash, ok := (*curr)[key]; ok {
			ret = append(ret, messageHash)
		}
	}
	return
}
