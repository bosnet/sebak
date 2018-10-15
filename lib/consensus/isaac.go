package consensus

import (
	"context"
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

type SyncController interface {
	SetSyncTargetBlock(ctx context.Context, height uint64, nodeAddrs []string) error
}

type ISAAC struct {
	sync.RWMutex

	latestConfirmedBlock block.Block
	connectionManager    network.ConnectionManager
	proposerSelector     ProposerSelector
	log                  logging.Logger
	policy               ballot.VotingThresholdPolicy
	nodesHeight          map[ /* Node.Address() */ string]uint64
	syncer               SyncController
	latestReqSyncHeight  uint64
	LatestBallot         ballot.Ballot

	NetworkID       []byte
	Node            *node.LocalNode
	TransactionPool *transaction.TransactionPool
	RunningRounds   map[ /* Round.Index() */ string]*RunningRound
	LatestRound     round.Round
}

// ISAAC should know network.ConnectionManager
// because the ISAAC uses connected validators when calculating proposer
func NewISAAC(networkID []byte, node *node.LocalNode, p ballot.VotingThresholdPolicy,
	cm network.ConnectionManager, syncer SyncController) (is *ISAAC, err error) {

	is = &ISAAC{
		NetworkID:         networkID,
		Node:              node,
		policy:            p,
		TransactionPool:   transaction.NewTransactionPool(),
		RunningRounds:     map[string]*RunningRound{},
		connectionManager: cm,
		proposerSelector:  SequentialSelector{cm},
		log:               log.New(logging.Ctx{"node": node.Alias()}),
		nodesHeight:       make(map[string]uint64),
		syncer:            syncer,
		LatestBallot:      ballot.Ballot{},
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

func (is *ISAAC) SetLatestConsensusedBlock(block block.Block) {
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

func (is *ISAAC) SaveNodeHeight(senderAddr string, height uint64) {
	is.nodesHeight[senderAddr] = height
}

func (is *ISAAC) IsAvailableRound(round round.Round) bool {
	if round.BlockHeight == is.latestConfirmedBlock.Height {
		if is.isInitRound(round) {
			return true
		}

		if round.BlockHash != is.latestConfirmedBlock.Hash {
			return false
		}

		if round.BlockHeight == is.LatestRound.BlockHeight {
			if round.Number <= is.LatestRound.Number {
				return false
			}
		}
		return true
	}

	return false
}

func (is *ISAAC) isInitRound(round round.Round) bool {
	genesisHeight := uint64(1)
	return is.LatestRound.BlockHash == "" && round.BlockHeight == genesisHeight
}

func (is *ISAAC) StartSync(height uint64, nodeAddrs []string) {
	is.log.Debug("begin is.StartSync")
	if is.syncer != nil && len(nodeAddrs) > 0 && is.latestReqSyncHeight < height {
		if is.Node.State() != node.StateSYNC {
			is.log.Info("node state transits to sync", "height", height)
			is.Node.SetSync()
		}
		is.latestReqSyncHeight = height
		is.log.Debug("before is.SetSyncTargetBlock")
		if err := is.syncer.SetSyncTargetBlock(context.Background(), height, nodeAddrs); err != nil {
			is.log.Error("syncer.SetSyncTargetBlock", "err", err, "height", height)
		}
	}

	return
}

func (is *ISAAC) GetSyncInfo() (uint64, []string, error) {
	is.log.Debug("begin is.GetSyncInfo", "is.nodesHeight", is.nodesHeight)

	overHeightCount := make(map[ /* height */ uint64] /* count */ int)
	for _, height := range is.nodesHeight {
		if _, ok := overHeightCount[height]; ok {
			overHeightCount[height]++
		} else {
			overHeightCount[height] = 1
		}
	}

	threshold := is.policy.Threshold()
	biggestHeight := uint64(1)

	for height, count := range overHeightCount {
		if count >= threshold {
			if height > biggestHeight {
				biggestHeight = height
			}
		}
	}

	if biggestHeight <= 1 {
		return 1, []string{}, nil
	}

	nodeAddrs := []string{}
	for nodeAddr, height := range is.nodesHeight {
		if height >= biggestHeight {
			nodeAddrs = append(nodeAddrs, nodeAddr)
		}
	}

	if len(nodeAddrs) == 0 {
		return 1, nodeAddrs, errors.New("biggestHeight nodeAddrs is empty")
	}

	return biggestHeight, nodeAddrs, nil
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

func (is *ISAAC) LatestConfirmedBlock() block.Block {
	is.RLock()
	defer is.RUnlock()
	return is.latestConfirmedBlock
}
