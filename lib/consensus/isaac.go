package consensus

import (
	"errors"
	"sync"

	logging "github.com/inconshreveable/log15"

	"boscoin.io/sebak/lib/ballot"
	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/consensus/round"
	"boscoin.io/sebak/lib/network"
	"boscoin.io/sebak/lib/node"
	"boscoin.io/sebak/lib/transaction"
)

type ISAAC struct {
	sync.RWMutex

	latestBlock       block.Block
	connectionManager network.ConnectionManager
	proposerSelector  ProposerSelector
	log               logging.Logger
	policy            ballot.VotingThresholdPolicy

	NetworkID       []byte
	Node            *node.LocalNode
	TransactionPool *transaction.Pool
	RunningRounds   map[ /* Round.Index() */ string]*RunningRound
	LatestRound     round.Round
}

// ISAAC should know network.ConnectionManager
// because the ISAAC uses connected validators when calculating proposer
func NewISAAC(networkID []byte, node *node.LocalNode, p ballot.VotingThresholdPolicy,
	cm network.ConnectionManager) (is *ISAAC, err error) {

	is = &ISAAC{
		NetworkID:         networkID,
		Node:              node,
		policy:            p,
		TransactionPool:   transaction.NewPool(),
		RunningRounds:     map[string]*RunningRound{},
		connectionManager: cm,
		proposerSelector:  SequentialSelector{cm},
		log:               log.New(logging.Ctx{"node": node.Alias()}),
	}

	return
}

func (is *ISAAC) CloseConsensus(proposer string, round round.Round, vh ballot.VotingHole) (err error) {
	is.Lock()
	defer is.Unlock()

	is.SetLatestRound(round)

	if vh == ballot.VotingNOTYET {
		err = errors.New("invalid VotingHole, `VotingNOTYET`")
		return
	}

	roundHash := round.Index()
	rr, found := is.RunningRounds[roundHash]
	if !found {
		return
	}

	if vh == ballot.VotingNO {
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

func (is *ISAAC) SetLatestBlock(block block.Block) {
	is.Lock()
	defer is.Unlock()
	is.latestBlock = block
}

func (is *ISAAC) SetLatestRound(round round.Round) {
	is.LatestRound = round
}

func (is *ISAAC) SetProposerSelector(p ProposerSelector) {
	is.proposerSelector = p
}

func (is *ISAAC) ConnectionManager() network.ConnectionManager {
	return is.connectionManager
}

func (is *ISAAC) SelectProposer(blockHeight uint64, roundNumber uint64) string {
	return is.proposerSelector.Select(blockHeight, roundNumber)
}

func (is *ISAAC) IsAvailableRound(round round.Round) bool {
	// check current round is from InitRound
	if is.LatestRound.BlockHash == "" {
		return true
	}

	if round.BlockHeight < is.latestBlock.Height {
		return false
	} else if round.BlockHeight == is.latestBlock.Height {
		if round.BlockHash != is.latestBlock.Hash {
			return false
		}
	} else {
		// TODO if incoming round.BlockHeight is bigger than
		// latestBlock.Height and this round confirmed successfully,
		// this node will get into sync state
	}

	if round.BlockHeight == is.LatestRound.BlockHeight {
		if round.Number <= is.LatestRound.Number {
			return false
		}
	}

	return true
}

func (is *ISAAC) IsVoted(b ballot.Ballot) bool {
	is.RLock()
	defer is.RUnlock()
	var found bool

	var runningRound *RunningRound
	if runningRound, found = is.RunningRounds[b.Round().Index()]; !found {
		return false
	}

	return runningRound.IsVoted(b)
}

func (is *ISAAC) Vote(b ballot.Ballot) (isNew bool, err error) {
	is.RLock()
	defer is.RUnlock()
	roundHash := b.Round().Index()

	var found bool
	var runningRound *RunningRound
	if runningRound, found = is.RunningRounds[roundHash]; !found {
		proposer := is.SelectProposer(
			b.Round().BlockHeight,
			b.Round().Number,
		)

		if runningRound, err = NewRunningRound(proposer, b); err != nil {
			return true, err
		}

		is.RunningRounds[roundHash] = runningRound
		isNew = true
	} else {
		if _, found = runningRound.Voted[b.Proposer()]; !found {
			isNew = true
		}

		runningRound.Vote(b)
	}

	return
}

func (is *ISAAC) CanGetVotingResult(b ballot.Ballot) (RoundVoteResult, ballot.VotingHole, bool) {
	is.RLock()
	defer is.RUnlock()
	runningRound, _ := is.RunningRounds[b.Round().Index()]
	if roundVote, err := runningRound.RoundVote(b.Proposer()); err == nil {
		return roundVote.CanGetVotingResult(is.policy, b.State(), is.log)
	} else {
		return nil, ballot.VotingNOTYET, false
	}
}

func (is *ISAAC) IsVotedByNode(b ballot.Ballot, node string) (bool, error) {
	is.RLock()
	defer is.RUnlock()
	runningRound, _ := is.RunningRounds[b.Round().Index()]
	if roundVote, err := runningRound.RoundVote(b.Proposer()); err == nil {
		return roundVote.IsVotedByNode(b.State(), node), nil
	} else {
		return false, err
	}
}

func (is *ISAAC) HasRunningRound(roundHash string) bool {
	is.RLock()
	defer is.RUnlock()
	_, found := is.RunningRounds[roundHash]
	return found
}

func (is *ISAAC) HasSameProposer(b ballot.Ballot) bool {
	is.RLock()
	defer is.RUnlock()
	if runningRound, found := is.RunningRounds[b.Round().Index()]; found {
		return runningRound.Proposer == b.Proposer()
	}

	return false
}

func (is *ISAAC) LatestBlock() block.Block {
	is.RLock()
	defer is.RUnlock()
	return is.latestBlock
}
