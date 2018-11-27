package consensus

import (
	logging "github.com/inconshreveable/log15"

	"boscoin.io/sebak/lib/ballot"
	"boscoin.io/sebak/lib/voting"
)

type RoundVoteResult map[ /* Node.Address() */ string]voting.Hole

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

func (rv *RoundVote) Vote(b ballot.Ballot) (isNew bool, err error) {
	if b.State() == ballot.StateSIGN || b.State() == ballot.StateACCEPT {
		result := rv.GetResult(b.State())

		_, isNew = result[b.Source()]
		result[b.Source()] = b.Vote()
	}

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

func (rv *RoundVote) CanGetVotingResult(policy voting.ThresholdPolicy, state ballot.State, log logging.Logger) (RoundVoteResult, voting.Hole, bool) {
	threshold := policy.Threshold()
	if threshold < 1 {
		return RoundVoteResult{}, voting.NOTYET, false
	}

	result := rv.GetResult(state)

	var yes, no, expired int
	for _, votingHole := range result {
		switch votingHole {
		case voting.YES:
			yes++
		case voting.NO:
			no++
		case voting.EXP:
			expired++
		}
	}

	log.Debug(
		"check threshold in isaac",
		"threshold", threshold,
		"yes", yes,
		"no", no,
		"expired", expired,
		"state", state,
	)

	if yes >= threshold {
		return result, voting.YES, true
	} else if no >= threshold {
		return result, voting.NO, true
	} else {
		// do nothing
	}

	// check draw!
	total := policy.Validators()
	voted := yes + no + expired
	if cannotBeOver(total-voted, threshold, yes, no) { // draw
		return result, voting.EXP, true
	}

	return result, voting.NOTYET, false
}

func cannotBeOver(remain, threshold, yes, no int) bool {
	return remain+yes < threshold && remain+no < threshold
}
