package sebak

import (
	"errors"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/error"
	"boscoin.io/sebak/lib/node"
)

type RoundVoteResult map[ /* Node.Address() */ string]sebakcommon.VotingHole

type RoundVote struct {
	SIGN   RoundVoteResult
	ACCEPT RoundVoteResult
}

func NewRoundVote(ballot Ballot) (rv *RoundVote) {
	rv = &RoundVote{
		SIGN:   RoundVoteResult{},
		ACCEPT: RoundVoteResult{},
	}

	rv.Vote(ballot)

	return rv
}

func (rv *RoundVote) IsVoted(ballot Ballot) bool {
	result := rv.GetResult(ballot.State())

	_, found := result[ballot.Source()]
	return found
}

func (rv *RoundVote) IsVotedByNode(state sebakcommon.BallotState, node string) bool {
	result := rv.GetResult(state)

	_, found := result[node]
	return found
}

func (rv *RoundVote) Vote(ballot Ballot) (isNew bool, err error) {
	if ballot.IsFromProposer() {
		return
	}

	result := rv.GetResult(ballot.State())
	_, isNew = result[ballot.Source()]
	result[ballot.Source()] = ballot.Vote()

	return
}

func (rv *RoundVote) GetResult(state sebakcommon.BallotState) (result RoundVoteResult) {
	if !state.IsValidForVote() {
		return
	}

	switch state {
	case sebakcommon.BallotStateSIGN:
		result = rv.SIGN
	case sebakcommon.BallotStateACCEPT:
		result = rv.ACCEPT
	}

	return result
}

func (rv *RoundVote) CanGetVotingResult(policy sebakcommon.VotingThresholdPolicy, state sebakcommon.BallotState) (RoundVoteResult, sebakcommon.VotingHole, bool) {
	threshold := policy.Threshold(state)
	if threshold < 1 {
		return RoundVoteResult{}, sebakcommon.VotingNOTYET, false
	}

	result := rv.GetResult(state)
	if len(result) < int(threshold) {
		return result, sebakcommon.VotingNOTYET, false
	}

	var yes, no int
	for _, votingHole := range result {
		switch votingHole {
		case sebakcommon.VotingYES:
			yes++
		case sebakcommon.VotingNO:
			no++
		}
	}

	log.Debug(
		"check threshold in isaac",
		"threshold", threshold,
		"yes", yes,
		"no", no,
		"policy", policy,
		"state", state,
	)

	if yes >= threshold {
		return result, sebakcommon.VotingYES, true
	} else if no >= threshold {
		return result, sebakcommon.VotingNO, true
	}

	// check draw!
	total := policy.Validators()
	voted := yes + no
	if total-voted < threshold-yes && total-voted < threshold-no { // draw
		return result, sebakcommon.VotingNO, true
	}

	return result, sebakcommon.VotingNOTYET, false
}

type RunningRound struct {
	sebakcommon.SafeLock

	Round        Round
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
		err = sebakerror.ErrorRoundVoteNotFound
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

type TransactionPool struct {
	sebakcommon.SafeLock

	Pool    map[ /* Transaction.GetHash() */ string]Transaction
	Hashes  []string // Transaction.GetHash()
	Sources map[ /* Transaction.Source() */ string]bool
}

func NewTransactionPool() *TransactionPool {
	return &TransactionPool{
		Pool:    map[string]Transaction{},
		Hashes:  []string{},
		Sources: map[string]bool{},
	}
}

func (tp *TransactionPool) Len() int {
	return len(tp.Hashes)
}

func (tp *TransactionPool) Has(hash string) bool {
	_, found := tp.Pool[hash]
	return found
}

func (tp *TransactionPool) Get(hash string) (tx Transaction, found bool) {
	tx, found = tp.Pool[hash]
	return
}

func (tp *TransactionPool) Add(tx Transaction) bool {
	if _, found := tp.Pool[tx.GetHash()]; found {
		return false
	}

	tp.Lock()
	defer tp.Unlock()

	tp.Pool[tx.GetHash()] = tx
	tp.Hashes = append(tp.Hashes, tx.GetHash())
	tp.Sources[tx.Source()] = true

	return true
}

func (tp *TransactionPool) Remove(hashes ...string) {
	if len(hashes) < 1 {
		return
	}

	tp.Lock()
	defer tp.Unlock()

	indices := map[int]int{}
	var max int
	for _, hash := range hashes {
		index, found := sebakcommon.InStringArray(tp.Hashes, hash)
		if !found {
			continue
		}
		indices[index] = 1
		if index > max {
			max = index
		}

		if tx, found := tp.Get(hash); found {
			delete(tp.Sources, tx.Source())
		}
	}

	var newHashes []string
	for i, hash := range tp.Hashes {
		if i > max {
			newHashes = append(newHashes, hash)
			continue
		}

		if _, found := indices[i]; !found {
			newHashes = append(newHashes, hash)
			continue
		}

		delete(tp.Pool, hash)
	}

	tp.Hashes = newHashes

	return
}

func (tp *TransactionPool) AvailableTransactions() []string {
	tp.Lock()
	defer tp.Unlock()

	if tp.Len() <= MaxTransactionsInBallot {
		return tp.Hashes
	}

	return tp.Hashes[:MaxTransactionsInBallot]
}

func (tp *TransactionPool) IsSameSource(source string) (found bool) {
	_, found = tp.Sources[source]

	return
}

type ISAAC struct {
	sebakcommon.SafeLock

	NetworkID             []byte
	Node                  *sebaknode.LocalNode
	VotingThresholdPolicy sebakcommon.VotingThresholdPolicy
	TransactionPool       *TransactionPool
	RunningRounds         map[ /* Round.Hash() */ string]*RunningRound
	LatestConfirmedBlock  Block
	LatestRound           Round
}

func NewISAAC(networkID []byte, node *sebaknode.LocalNode, votingThresholdPolicy sebakcommon.VotingThresholdPolicy) (is *ISAAC, err error) {
	is = &ISAAC{
		NetworkID: networkID,
		Node:      node,
		VotingThresholdPolicy: votingThresholdPolicy,
		TransactionPool:       NewTransactionPool(),
		RunningRounds:         map[string]*RunningRound{},
	}

	return
}

func (is *ISAAC) CloseConsensus(proposer string, round Round, vh sebakcommon.VotingHole) (err error) {
	is.Lock()
	defer is.Unlock()

	if vh == sebakcommon.VotingNOTYET {
		err = errors.New("invalid VotingHole, `VotingNOTYET`")
		return
	}

	roundHash := round.Hash()
	rr, found := is.RunningRounds[roundHash]
	if !found {
		return
	}

	if vh == sebakcommon.VotingNO {
		delete(rr.Transactions, proposer)
		delete(rr.Voted, proposer)

		return
	}

	is.TransactionPool.Remove(rr.Transactions[proposer]...)

	delete(is.RunningRounds, roundHash)

	// remove all the same rounds
	for hash, runningRound := range is.RunningRounds {
		if runningRound.Round.BlockHeight > round.BlockHeight {
			continue
		}
		delete(is.RunningRounds, hash)
	}

	return
}

func (is *ISAAC) SetLatestConsensusedBlock(block Block) {
	is.LatestConfirmedBlock = block
}

func (is *ISAAC) SetLatestRound(round Round) {
	is.LatestRound = round
}

func (is *ISAAC) IsAvailableRound(round Round) bool {
	// check current round is from InitRound
	if is.LatestRound.BlockHash == "" {
		return true
	}

	if round.BlockHeight < is.LatestConfirmedBlock.Height {
		return false
	} else if round.BlockHeight == is.LatestConfirmedBlock.Height {
		if round.BlockHash != is.LatestConfirmedBlock.Hash {
			return false
		}
	} else {
		// TODO if incoming round.BlockHeight is bigger than
		// LatestConfirmedBlock.Height and this round confirmed successfully,
		// this node will get into catchup status
	}

	if round.BlockHeight == is.LatestRound.BlockHeight {
		if round.Number <= is.LatestRound.Number {
			return false
		}
	}

	return true
}
