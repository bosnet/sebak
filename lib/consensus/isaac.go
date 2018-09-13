package consensus

import (
	"errors"
	"sync"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/consensus/round"
	"boscoin.io/sebak/lib/network"
	"boscoin.io/sebak/lib/node"
	"boscoin.io/sebak/lib/transaction"
)

type ISAAC struct {
	sync.RWMutex

	NetworkID             []byte
	Node                  *node.LocalNode
	VotingThresholdPolicy common.VotingThresholdPolicy
	TransactionPool       *transaction.TransactionPool
	RunningRounds         map[ /* Round.Hash() */ string]*RunningRound
	latestConfirmedBlock  block.Block
	LatestRound           round.Round
	connectionManager     *network.ConnectionManager
}

// ISAAC should know network.ConnectionManager
// because the ISAAC uses connected validators when calculating proposer
func NewISAAC(networkID []byte, node *node.LocalNode, policy common.VotingThresholdPolicy,
	cm *network.ConnectionManager) (is *ISAAC, err error) {

	is = &ISAAC{
		NetworkID: networkID, Node: node,
		VotingThresholdPolicy: policy,
		TransactionPool:       transaction.NewTransactionPool(),
		RunningRounds:         map[string]*RunningRound{},
		connectionManager:     cm,
	}

	is.connectionManager.SetBroadcaster(network.NewSimpleBroadcaster(is.ConnectionManager()))

	return
}

func (is *ISAAC) CloseConsensus(proposer string, round round.Round, vh common.VotingHole) (err error) {
	is.Lock()
	defer is.Unlock()

	is.SetLatestRound(round)

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

func (is *ISAAC) SetLatestConsensusedBlock(block block.Block) {
	is.latestConfirmedBlock = block
}

func (is *ISAAC) SetLatestRound(round round.Round) {
	is.LatestRound = round
}

func (is *ISAAC) SetBroadcaster(b network.Broadcaster) {
	is.connectionManager.SetBroadcaster(b)
}

func (is *ISAAC) ConnectionManager() *network.ConnectionManager {
	return is.connectionManager
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

func (is *ISAAC) IsVoted(b block.Ballot) bool {
	is.RLock()
	defer is.RUnlock()
	var found bool

	var runningRound *RunningRound
	if runningRound, found = is.RunningRounds[b.Round().Hash()]; !found {
		return false
	}

	return runningRound.IsVoted(b)
}

func (is *ISAAC) Vote(b block.Ballot) (isNew bool, err error) {
	is.RLock()
	defer is.RUnlock()
	roundHash := b.Round().Hash()

	var found bool
	var runningRound *RunningRound
	if runningRound, found = is.RunningRounds[roundHash]; !found {
		proposer := is.ConnectionManager().CalculateProposer(
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
func (is *ISAAC) CanGetVotingResult(b block.Ballot) (RoundVoteResult, common.VotingHole, bool) {
	is.RLock()
	defer is.RUnlock()
	runningRound, _ := is.RunningRounds[b.Round().Hash()]
	if roundVote, err := runningRound.RoundVote(b.Proposer()); err == nil {
		return roundVote.CanGetVotingResult(is.VotingThresholdPolicy, b.State())
	} else {
		return nil, common.VotingNOTYET, false
	}
}

func (is *ISAAC) IsVotedByNode(b block.Ballot, node string) (bool, error) {
	is.RLock()
	defer is.RUnlock()
	runningRound, _ := is.RunningRounds[b.Round().Hash()]
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

func (is *ISAAC) HasSameProposer(b block.Ballot) bool {
	is.RLock()
	defer is.RUnlock()
	if runningRound, found := is.RunningRounds[b.Round().Hash()]; found {
		return runningRound.Proposer == b.Proposer()
	}

	return false
}

func (is *ISAAC) AddRunningRound(roundHash string, runningRound *RunningRound) {
	is.Lock()
	defer is.Unlock()
	is.RunningRounds[roundHash] = runningRound
}

func (is *ISAAC) LatestConfirmedBlock() block.Block {
	is.RLock()
	defer is.RUnlock()
	return is.latestConfirmedBlock
}
