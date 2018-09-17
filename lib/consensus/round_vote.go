package consensus

import (
	"boscoin.io/sebak/lib/ballot"
)

type RoundVoteResult map[ /* Node.Address() */ string]ballot.VotingHole

type RoundVote struct {
	SIGN   RoundVoteResult
	ACCEPT RoundVoteResult
}

func NewRoundVote(ballot ballot.Ballot) (rv *RoundVote) {
	rv = &RoundVote{
		SIGN:   RoundVoteResult{},
		ACCEPT: RoundVoteResult{},
	}

	rv.Vote(ballot)

	return rv
}

func (rv *RoundVote) IsVoted(ballot ballot.Ballot) bool {
	result := rv.GetResult(ballot.State())

	_, found := result[ballot.Source()]
	return found
}

func (rv *RoundVote) IsVotedByNode(state ballot.State, node string) bool {
	result := rv.GetResult(state)

	_, found := result[node]
	return found
}

func (rv *RoundVote) Vote(ballot ballot.Ballot) (isNew bool, err error) {
	if ballot.IsFromProposer() {
		return
	}

	result := rv.GetResult(ballot.State())
	_, isNew = result[ballot.Source()]
	result[ballot.Source()] = ballot.Vote()

	return
}

func (rv *RoundVote) GetResult(state ballot.State) (result RoundVoteResult) {
	if !state.IsValidForVote() {
		return
	}

	switch state {
	case ballot.StateSIGN:
		result = rv.SIGN
	case ballot.StateACCEPT:
		result = rv.ACCEPT
	}

	return result
}

func (rv *RoundVote) CanGetVotingResult(policy ballot.VotingThresholdPolicy, state ballot.State) (RoundVoteResult, ballot.VotingHole, bool) {
	threshold := policy.Threshold(state)
	if threshold < 1 {
		return RoundVoteResult{}, ballot.VotingNOTYET, false
	}

	result := rv.GetResult(state)
	if len(result) < int(threshold) {
		return result, ballot.VotingNOTYET, false
	}

	var yes, no, expired int
	for _, votingHole := range result {
		switch votingHole {
		case ballot.VotingYES:
			yes++
		case ballot.VotingNO:
			no++
		case ballot.VotingEXP:
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

	if state == ballot.StateSIGN {
		if yes >= threshold {
			return result, ballot.VotingYES, true
		} else if no >= threshold+1 {
			return result, ballot.VotingNO, true
		}
	} else if state == ballot.StateACCEPT {
		if yes >= threshold {
			return result, ballot.VotingYES, true
		} else if no >= threshold {
			return result, ballot.VotingNO, true
		}
	} else {
		// do nothing
	}

	// check draw!
	total := policy.Validators()
	voted := yes + no + expired
	if cannotBeOver(total-voted, threshold, yes, no) { // draw
		return result, ballot.VotingEXP, true
	}

	return result, ballot.VotingNOTYET, false
}

func cannotBeOver(remain, threshold, yes, no int) bool {
	return remain+yes < threshold && remain+no < threshold
}
