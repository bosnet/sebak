package consensus

import (
	logging "github.com/inconshreveable/log15"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/common"
)

type RoundVoteResult map[ /* Node.Address() */ string]common.VotingHole

type RoundVote struct {
	SIGN   RoundVoteResult
	ACCEPT RoundVoteResult
}

func NewRoundVote(ballot block.Ballot) (rv *RoundVote) {
	rv = &RoundVote{
		SIGN:   RoundVoteResult{},
		ACCEPT: RoundVoteResult{},
	}

	rv.Vote(ballot)

	return rv
}

func (rv *RoundVote) IsVoted(ballot block.Ballot) bool {
	result := rv.GetResult(ballot.State())

	_, found := result[ballot.Source()]
	return found
}

func (rv *RoundVote) IsVotedByNode(state common.BallotState, node string) bool {
	result := rv.GetResult(state)

	_, found := result[node]
	return found
}

func (rv *RoundVote) Vote(ballot block.Ballot) (isNew bool, err error) {
	if ballot.IsFromProposer() {
		return
	}

	result := rv.GetResult(ballot.State())
	_, isNew = result[ballot.Source()]
	result[ballot.Source()] = ballot.Vote()

	return
}

func (rv *RoundVote) GetResult(state common.BallotState) (result RoundVoteResult) {
	if !state.IsValidForVote() {
		return
	}

	switch state {
	case common.BallotStateSIGN:
		result = rv.SIGN
	case common.BallotStateACCEPT:
		result = rv.ACCEPT
	}

	return result
}

func (rv *RoundVote) CanGetVotingResult(policy common.VotingThresholdPolicy, state common.BallotState, log logging.Logger) (RoundVoteResult, common.VotingHole, bool) {
	threshold := policy.Threshold(state)
	if threshold < 1 {
		return RoundVoteResult{}, common.VotingNOTYET, false
	}

	result := rv.GetResult(state)
	if len(result) < int(threshold) {
		return result, common.VotingNOTYET, false
	}

	var yes, no, expired int
	for _, votingHole := range result {
		switch votingHole {
		case common.VotingYES:
			yes++
		case common.VotingNO:
			no++
		case common.VotingEXP:
			expired++
		}
	}

	log.Debug(
		"check threshold in isaac",
		"threshold", threshold,
		"yes", yes,
		"no", no,
		"expired", expired,
		"policy", policy,
		"state", state,
	)

	if state == common.BallotStateSIGN {
		if yes >= threshold {
			return result, common.VotingYES, true
		} else if no >= threshold+1 {
			return result, common.VotingNO, true
		}
	} else if state == common.BallotStateACCEPT {
		if yes >= threshold {
			return result, common.VotingYES, true
		} else if no >= threshold {
			return result, common.VotingNO, true
		}
	} else {
		// do nothing
	}

	// check draw!
	total := policy.Validators()
	voted := yes + no + expired
	if cannotBeOver(total-voted, threshold, yes, no) { // draw
		return result, common.VotingEXP, true
	}

	return result, common.VotingNOTYET, false
}

func cannotBeOver(remain, threshold, yes, no int) bool {
	return remain+yes < threshold && remain+no < threshold
}
