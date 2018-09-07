package sebak

import (
	"errors"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/node"
	"boscoin.io/sebak/lib/round"
)

type TransactionPool struct {
	common.SafeLock

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
		index, found := common.InStringArray(tp.Hashes, hash)
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

func (tp *TransactionPool) AvailableTransactions(conf *IsaacConfiguration) []string {
	tp.Lock()
	defer tp.Unlock()

	if tp.Len() <= int(conf.TransactionsLimit) {
		return tp.Hashes
	}

	return tp.Hashes[:conf.TransactionsLimit]
}

func (tp *TransactionPool) IsSameSource(source string) (found bool) {
	_, found = tp.Sources[source]

	return
}

type ISAAC struct {
	common.SafeLock

	NetworkID             []byte
	Node                  *node.LocalNode
	VotingThresholdPolicy common.VotingThresholdPolicy
	TransactionPool       *TransactionPool
	RunningRounds         map[ /* Round.Hash() */ string]*RunningRound
	LatestConfirmedBlock  Block
	LatestRound           round.Round
}

func NewISAAC(networkID []byte, node *node.LocalNode, votingThresholdPolicy common.VotingThresholdPolicy) (is *ISAAC, err error) {
	is = &ISAAC{
		NetworkID:             networkID,
		Node:                  node,
		VotingThresholdPolicy: votingThresholdPolicy,
		TransactionPool:       NewTransactionPool(),
		RunningRounds:         map[string]*RunningRound{},
	}

	return
}

func (is *ISAAC) CloseConsensus(proposer string, round round.Round, vh common.VotingHole) (err error) {
	is.Lock()
	defer is.Unlock()

	if vh == common.VotingNOTYET {
		err = errors.New("invalid VotingHole, `VotingNOTYET`")
		return
	}

	roundHash := round.Hash()
	rr, found := is.RunningRounds[roundHash]
	if !found {
		return
	}

	if vh == common.VotingNO {
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

func (is *ISAAC) SetLatestRound(round round.Round) {
	is.LatestRound = round
}

func (is *ISAAC) IsAvailableRound(round round.Round) bool {
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
		// this node will get into sync state
	}

	if round.BlockHeight == is.LatestRound.BlockHeight {
		if round.Number <= is.LatestRound.Number {
			return false
		}
	}

	return true
}
