package sebak

import (
	"errors"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/error"
	"boscoin.io/sebak/lib/node"
)

type RoundVote map[ /* Node.Address() */ string]VotingHole

func (rv RoundVote) CanGetVotingResult(policy sebakcommon.VotingThresholdPolicy) (VotingHole, bool) {
	threshold := policy.Threshold()
	if threshold < 1 {
		return VotingNOTYET, false
	}
	if len(rv) < int(threshold) {
		return VotingNOTYET, false
	}

	var yes int
	var no int
	for _, vh := range rv {
		if vh == VotingYES {
			yes++
		} else if vh == VotingNO {
			no++
		}
	}

	log.Debug(
		"check threshold in isaac",
		"threshold", threshold,
		"yes", yes,
		"no", no,
		"policy", policy,
	)

	if yes >= threshold {
		return VotingYES, true
	} else if no >= threshold {
		return VotingNO, true
	}

	// check draw!
	total := policy.Validators()
	voted := yes + no
	if total-voted < threshold-yes && total-voted < threshold-no { // draw
		return VotingNO, true
	}

	return VotingNOTYET, false
}

type RunningRound struct {
	sebakcommon.SafeLock

	Round        Round
	Proposer     string                              // LocalNode's `Proposer`
	Transactions map[ /* Proposer */ string][]string /* Transaction.Hash */
	Voted        map[ /* Proposer */ string]RoundVote
}

func NewRunningRound(proposer string, rb Ballot) *RunningRound {
	transactions := map[string][]string{
		rb.Proposer(): rb.Transactions(),
	}

	voted := map[string]RoundVote{
		rb.Proposer(): RoundVote{
			rb.Source(): rb.Vote(),
		},
	}

	if !rb.IsFromProposer() {
		voted[rb.Proposer()][rb.Proposer()] = VotingYES
	}

	return &RunningRound{
		Round:        rb.Round(),
		Proposer:     proposer,
		Transactions: transactions,
		Voted:        voted,
	}
}

func (rr *RunningRound) RoundVote(proposer string) (rv RoundVote, err error) {
	var found bool
	rv, found = rr.Voted[proposer]
	if !found {
		err = sebakerror.ErrorRoundVoteNotFound
		return
	}
	return
}

func (rr *RunningRound) IsVoted(rb Ballot) bool {
	roundVote, err := rr.RoundVote(rb.Proposer())
	if err != nil {
		return false
	}

	_, voted := roundVote[rb.Source()]
	return voted
}

func (rr *RunningRound) Vote(rb Ballot) (isNew bool) {
	rr.Lock()
	defer rr.Unlock()

	if _, found := rr.Voted[rb.Proposer()]; !found {
		rr.Voted[rb.Proposer()] = RoundVote{}
		isNew = true
	}

	rr.Voted[rb.Proposer()][rb.Source()] = rb.Vote()
	return
}

type TransactionPool struct {
	sebakcommon.SafeLock

	Pool   map[ /* Transaction.GetHash() */ string]Transaction
	Hashes []string // Transaction.GetHash()
}

func NewTransactionPool() *TransactionPool {
	return &TransactionPool{
		Pool:   map[string]Transaction{},
		Hashes: []string{},
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

	return true
}

func (tp *TransactionPool) Remove(hashes ...string) {
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

func (is *ISAAC) IsRunningRound(roundNumber uint64) bool {
	for _, runningRound := range is.RunningRounds {
		if runningRound.Round.Number == roundNumber {
			return true
		}
	}
	return false
}

func (is *ISAAC) ReceiveMessage(m sebakcommon.Message) (err error) {
	if is.TransactionPool.Has(m.GetHash()) {
		err = sebakerror.ErrorNewButKnownMessage
		return
	}

	is.TransactionPool.Add(m.(Transaction))

	return
}

func (is *ISAAC) CloseConsensus(proposer string, round Round, vh VotingHole) (err error) {
	is.Lock()
	defer is.Unlock()

	if vh == VotingNOTYET {
		err = errors.New("invalid VotingHole, `VotingNOTYET`")
		return
	}

	roundHash := round.Hash()
	rr, found := is.RunningRounds[roundHash]
	if !found {
		return
	}

	if vh == VotingNO {
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
	if round.BlockHeight != is.LatestConfirmedBlock.Height {
		return false
	}

	if is.LatestRound.BlockHash == "" {
		return true
	}

	if round.BlockHeight == is.LatestRound.BlockHeight {
		if round.Number <= is.LatestRound.Number {
			return false
		}
	}

	return true
}
