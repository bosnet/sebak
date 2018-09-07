package sebak

import (
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/error"
	"boscoin.io/sebak/lib/round"
)

type RunningRound struct {
	common.SafeLock

	Round        round.Round
	Proposer     string                              // LocalNode's `Proposer`
	Transactions map[ /* Proposer */ string][]string /* Transaction.Hash */
	Voted        map[ /* Proposer */ string]*RoundVote
}

func NewRunningRound(proposer string, ballot Ballot) (*RunningRound, error) {
	transactions := map[string][]string{
		ballot.Proposer(): ballot.Transactions(),
	}

	roundVote := NewRoundVote(ballot)
	voted := map[string]*RoundVote{
		ballot.Proposer(): roundVote,
	}

	return &RunningRound{
		Round:        ballot.Round(),
		Proposer:     proposer,
		Transactions: transactions,
		Voted:        voted,
	}, nil
}

func (rr *RunningRound) RoundVote(proposer string) (rv *RoundVote, err error) {
	var found bool
	rv, found = rr.Voted[proposer]
	if !found {
		err = errors.ErrorRoundVoteNotFound
		return
	}
	return
}

func (rr *RunningRound) IsVoted(ballot Ballot) bool {
	roundVote, err := rr.RoundVote(ballot.Proposer())
	if err != nil {
		return false
	}

	return roundVote.IsVoted(ballot)
}

func (rr *RunningRound) Vote(ballot Ballot) {
	rr.Lock()
	defer rr.Unlock()

	if _, found := rr.Voted[ballot.Proposer()]; !found {
		rr.Voted[ballot.Proposer()] = NewRoundVote(ballot)
	} else {
		rr.Voted[ballot.Proposer()].Vote(ballot)
	}
}
