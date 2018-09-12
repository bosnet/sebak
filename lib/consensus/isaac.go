package consensus

import (
	"errors"
	"sync"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/consensus/round"
	"boscoin.io/sebak/lib/node"
	"boscoin.io/sebak/lib/transaction"
)

type ISAAC struct {
	sync.Mutex

	NetworkID             []byte
	Node                  *node.LocalNode
	VotingThresholdPolicy common.VotingThresholdPolicy
	TransactionPool       *transaction.TransactionPool
	RunningRounds         map[ /* Round.Hash() */ string]*RunningRound
	LatestConfirmedBlock  block.Block
	LatestRound           round.Round
}

func NewISAAC(networkID []byte, node *node.LocalNode, votingThresholdPolicy common.VotingThresholdPolicy) (is *ISAAC, err error) {
	is = &ISAAC{
		NetworkID: networkID, Node: node,
		VotingThresholdPolicy: votingThresholdPolicy,
		TransactionPool:       transaction.NewTransactionPool(),
		RunningRounds:         map[string]*RunningRound{},
	}

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
