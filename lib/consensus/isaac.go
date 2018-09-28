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

	latestConfirmedBlock block.Block
	connectionManager    network.ConnectionManager
	proposerSelector     ProposerSelector
	log                  logging.Logger
	policy               ballot.VotingThresholdPolicy

	NetworkID       []byte
	Node            *node.LocalNode
	TransactionPool *transaction.TransactionPool
	RunningRounds   map[ /* Round.BlockHeight */ uint64]*RunningRound
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
		TransactionPool:   transaction.NewTransactionPool(),
		RunningRounds:     map[uint64]*RunningRound{},
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

	rr, found := is.RunningRounds[round.BlockHeight]
	if !found {
		return
	}

	if vh == ballot.VotingNO {
		delete(rr.Transactions, proposer)
		delete(rr.Voted, proposer)

		return
	}

	is.TransactionPool.Remove(rr.Transactions[proposer]...)

	delete(is.RunningRounds, round.BlockHeight)

	// remove all the same rounds
	for hash, runningRound := range is.RunningRounds {
		if runningRound.Round.BlockHeight > round.BlockHeight {
			continue
		}
		delete(is.RunningRounds, hash)
	}

	return
}

func (is *ISAAC) SetLatestConfirmedBlock(block block.Block) {
	is.Lock()
	defer is.Unlock()
	is.latestConfirmedBlock = block
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

	if round.BlockHeight < is.latestConfirmedBlock.Height {
		return false
	} else if round.BlockHeight == is.latestConfirmedBlock.Height {
		if round.BlockHash != is.latestConfirmedBlock.Hash {
			return false
		}
	} else {
		// TODO if incoming round.BlockHeight is bigger than
		// latestConfirmedBlock.Height and this round confirmed successfully,
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
	if runningRound, found = is.RunningRounds[b.Round().BlockHeight]; !found {
		return false
	}

	return runningRound.IsVoted(b)
}

func (is *ISAAC) Vote(b ballot.Ballot) (isNew bool, err error) {
	is.RLock()
	defer is.RUnlock()
	blockHeight := b.Round().BlockHeight

	var found bool
	var runningRound *RunningRound
	if runningRound, found = is.RunningRounds[blockHeight]; !found {
		proposer := is.SelectProposer(
			b.Round().BlockHeight,
			b.Round().Number,
		)

		if runningRound, err = NewRunningRound(proposer, b); err != nil {
			return true, err
		}

		is.RunningRounds[blockHeight] = runningRound
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
	runningRound, _ := is.RunningRounds[b.Round().BlockHeight]
	if roundVote, err := runningRound.RoundVote(b.Proposer()); err == nil {
		return roundVote.CanGetVotingResult(is.policy, b.State(), is.log)
	} else {
		return nil, ballot.VotingNOTYET, false
	}
}

func (is *ISAAC) IsVotedByNode(b ballot.Ballot, node string) (bool, error) {
	is.RLock()
	defer is.RUnlock()
	runningRound, _ := is.RunningRounds[b.Round().BlockHeight]
	if roundVote, err := runningRound.RoundVote(b.Proposer()); err == nil {
		return roundVote.IsVotedByNode(b.State(), node), nil
	} else {
		return false, err
	}
}

func (is *ISAAC) HasRunningRound(blockHeight uint64) bool {
	is.RLock()
	defer is.RUnlock()
	_, found := is.RunningRounds[blockHeight]
	return found
}

func (is *ISAAC) HasSameProposer(b ballot.Ballot) bool {
	is.RLock()
	defer is.RUnlock()
	if runningRound, found := is.RunningRounds[b.Round().BlockHeight]; found {
		return runningRound.Proposer == b.Proposer()
	}

	return false
}

func (is *ISAAC) AddRunningRound(blockHeight uint64, theBallot ballot.Ballot) error {
	is.Lock()
	defer is.Unlock()
	runningRound, err := NewRunningRound(is.Node.Address(), theBallot)
	if err != nil {
		return err
	}
	is.RunningRounds[blockHeight] = runningRound
	log.Debug("ballot broadcasted and voted", "runningRound", runningRound)

	return err
}

func (is *ISAAC) LatestConfirmedBlock() block.Block {
	is.RLock()
	defer is.RUnlock()
	return is.latestConfirmedBlock
}
