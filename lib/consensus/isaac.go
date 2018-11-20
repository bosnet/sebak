package consensus

import (
	"context"
	"errors"
	"fmt"
	"sync"

	logging "github.com/inconshreveable/log15"

	"boscoin.io/sebak/lib/ballot"
	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/network"
	"boscoin.io/sebak/lib/node"
	"boscoin.io/sebak/lib/storage"
	"boscoin.io/sebak/lib/voting"
)

type SyncController interface {
	SetSyncTargetBlock(ctx context.Context, height uint64, nodeAddrs []string) error
}

type ISAAC struct {
	sync.RWMutex

	latestBlock         block.Block
	connectionManager   network.ConnectionManager
	storage             *storage.LevelDBBackend
	proposerSelector    ProposerSelector
	log                 logging.Logger
	policy              voting.ThresholdPolicy
	nodesHeight         map[ /* Node.Address() */ string]uint64
	syncer              SyncController
	latestReqSyncHeight uint64

	LatestBallot      ballot.Ballot
	NetworkID         []byte
	Node              *node.LocalNode
	RunningRounds     map[ /* Round.Index() */ string]*RunningRound
	LatestVotingBasis voting.Basis
	Conf              common.Config
}

// ISAAC should know network.ConnectionManager
// because the ISAAC uses connected validators when calculating proposer
func NewISAAC(node *node.LocalNode, p voting.ThresholdPolicy,
	cm network.ConnectionManager, st *storage.LevelDBBackend, conf common.Config, syncer SyncController) (is *ISAAC, err error) {

	is = &ISAAC{
		NetworkID:         conf.NetworkID,
		Node:              node,
		policy:            p,
		RunningRounds:     map[string]*RunningRound{},
		connectionManager: cm,
		storage:           st,
		proposerSelector:  SequentialSelector{cm},
		Conf:              conf,
		log:               log.New(logging.Ctx{"node": node.Alias()}),
		nodesHeight:       make(map[string]uint64),
		syncer:            syncer,
		LatestBallot:      ballot.Ballot{},
	}

	return
}

func (is *ISAAC) SetLatestVotingBasis(basis voting.Basis) {
	is.Lock()
	defer is.Unlock()

	is.LatestVotingBasis = basis
}

func (is *ISAAC) SetProposerSelector(p ProposerSelector) {
	is.proposerSelector = p
}

func (is *ISAAC) ConnectionManager() network.ConnectionManager {
	return is.connectionManager
}

func (is *ISAAC) SelectProposer(blockHeight uint64, round uint64) string {
	return is.proposerSelector.Select(blockHeight, round)
}

func (is *ISAAC) SaveNodeHeight(senderAddr string, height uint64) {
	is.Lock()
	defer is.Unlock()

	is.nodesHeight[senderAddr] = height
}

func (is *ISAAC) IsValidVotingBasis(basis voting.Basis, latestBlock block.Block) bool {
	is.RLock()
	defer is.RUnlock()

	if basis.Height == latestBlock.Height {
		if is.isInitRound(basis) {
			return true
		}

		if basis.BlockHash != latestBlock.Hash {
			return false
		}

		if basis.Height == is.LatestVotingBasis.Height {
			if basis.Round <= is.LatestVotingBasis.Round {
				return false
			}
		}
		return true
	}

	return false
}

func (is *ISAAC) isInitRound(basis voting.Basis) bool {
	return is.LatestVotingBasis.BlockHash == "" && basis.Height == common.GenesisBlockHeight
}

func (is *ISAAC) StartSync(height uint64, nodeAddrs []string) {
	is.log.Debug("begin ISAAC.StartSync")
	if is.syncer == nil || len(nodeAddrs) < 1 || is.latestReqSyncHeight >= height {
		return
	}
	if is.Node.State() != node.StateSYNC {
		is.log.Info("node state transits to sync", "height", height)
		is.Node.SetSync()
	}
	is.latestReqSyncHeight = height
	if err := is.syncer.SetSyncTargetBlock(context.Background(), height, nodeAddrs); err != nil {
		is.log.Error("syncer.SetSyncTargetBlock", "err", err, "height", height)
	}

	return
}

// GetSyncInfo gets the height it needs to sync.
// It returns height, node list and error.
// The height is the smallest height above the threshold.
// The node list is the nodes that sent the ballot when the threshold is exceeded.
func (is *ISAAC) GetSyncInfo() (uint64, []string, error) {
	is.log.Debug("begin ISAAC.GetSyncInfo", "is.nodesHeight", is.nodesHeight)
	threshold := is.policy.Threshold()
	if len(is.nodesHeight) < threshold {
		return 1, []string{}, errors.New(fmt.Sprintf("could not find enough nodes (threshold=%d) above", threshold))
	}

	var nodesHeight []common.KV
	for k, v := range is.nodesHeight {
		nodesHeight = append(nodesHeight, common.KV{Key: k, Value: v})
	}

	common.SortDecByValue(nodesHeight)

	height := nodesHeight[threshold-1].Value

	nodeAddrs := []string{}
	for _, kv := range nodesHeight[:threshold] {
		nodeAddrs = append(nodeAddrs, kv.Key)
	}

	return height, nodeAddrs, nil
}

func (is *ISAAC) IsVoted(b ballot.Ballot) bool {
	is.RLock()
	defer is.RUnlock()
	var found bool

	var runningRound *RunningRound
	if runningRound, found = is.RunningRounds[b.VotingBasis().Index()]; !found {
		return false
	}

	return runningRound.IsVoted(b)
}

func (is *ISAAC) Vote(b ballot.Ballot) (isNew bool, err error) {
	is.Lock()
	defer is.Unlock()
	basisIndex := b.VotingBasis().Index()

	var found bool
	var runningRound *RunningRound
	if runningRound, found = is.RunningRounds[basisIndex]; !found {
		proposer := is.SelectProposer(
			b.VotingBasis().Height,
			b.VotingBasis().Round,
		)

		if runningRound, err = NewRunningRound(proposer, b); err != nil {
			return true, err
		}

		is.RunningRounds[basisIndex] = runningRound
		isNew = true
	} else {
		if _, found = runningRound.Voted[b.Proposer()]; !found {
			isNew = true
		}

		runningRound.Vote(b)
	}

	return
}

func (is *ISAAC) CanGetVotingResult(b ballot.Ballot) (RoundVoteResult, voting.Hole, bool) {
	is.RLock()
	defer is.RUnlock()
	runningRound, _ := is.RunningRounds[b.VotingBasis().Index()]
	if roundVote, err := runningRound.RoundVote(b.Proposer()); err == nil {
		return roundVote.CanGetVotingResult(is.policy, b.State(), is.log)
	} else {
		return nil, voting.NOTYET, false
	}
}

func (is *ISAAC) IsVotedByNode(b ballot.Ballot, node string) (bool, error) {
	is.RLock()
	defer is.RUnlock()
	runningRound, _ := is.RunningRounds[b.VotingBasis().Index()]
	if roundVote, err := runningRound.RoundVote(b.Proposer()); err == nil {
		return roundVote.IsVotedByNode(b.State(), node), nil
	} else {
		return false, err
	}
}

func (is *ISAAC) HasRunningRound(basisIndex string) bool {
	is.RLock()
	defer is.RUnlock()
	_, found := is.RunningRounds[basisIndex]
	return found
}

func (is *ISAAC) HasSameProposer(b ballot.Ballot) bool {
	is.RLock()
	defer is.RUnlock()
	if runningRound, found := is.RunningRounds[b.VotingBasis().Index()]; found {
		return runningRound.Proposer == b.Proposer()
	}

	return false
}

func (is *ISAAC) LatestBlock() block.Block {
	return block.GetLatestBlock(is.storage)
}

func (is *ISAAC) RemoveRunningRoundsLowerOrEqualHeight(height uint64) {
	for hash, runningRound := range is.RunningRounds {
		if runningRound.VotingBasis.Height > height {
			continue
		}

		is.log.Debug("remove running rounds lower than or equal to height", "votingBasis", runningRound.VotingBasis)

		delete(runningRound.Transactions, runningRound.Proposer)
		delete(runningRound.Voted, runningRound.Proposer)
		delete(is.RunningRounds, hash)
	}
}

func (is *ISAAC) RemoveRunningRoundsLowerOrEqualBasis(basis voting.Basis) {
	for hash, runningRound := range is.RunningRounds {
		if runningRound.VotingBasis.Height > basis.Height {
			continue
		}

		if runningRound.VotingBasis.Height == basis.Height &&
			runningRound.VotingBasis.Round > basis.Round {
			continue
		}

		is.log.Debug("remove running rounds lower than or equal to basis", "votingBasis", runningRound.VotingBasis)

		delete(runningRound.Transactions, runningRound.Proposer)
		delete(runningRound.Voted, runningRound.Proposer)
		delete(is.RunningRounds, hash)
	}
}

func (is *ISAAC) RemoveRunningRoundsExceptExpired(state ISAACState) {
	is.log.Debug("remove running rounds except expired", "ISAACState", state)
	if (state.BallotState != ballot.StateSIGN) &&
		(state.BallotState != ballot.StateACCEPT) {
		return
	}
	for _, runningRound := range is.RunningRounds {
		if runningRound.VotingBasis.Height != state.Height ||
			runningRound.VotingBasis.Round != state.Round {
			continue
		}

		delete(runningRound.Transactions, runningRound.Proposer)
		for _, roundVote := range runningRound.Voted {
			var votingResults RoundVoteResult
			if state.BallotState == ballot.StateSIGN {
				votingResults = roundVote.SIGN
			} else {
				votingResults = roundVote.ACCEPT
			}

			removeTargets := []string{}
			for source, vote := range votingResults {
				if vote != voting.EXP {
					removeTargets = append(removeTargets, source)
				}
			}

			is.log.Debug("remove expired results(YES or NO)", "targets", removeTargets, "voting-results", votingResults)
			for _, source := range removeTargets {
				delete(votingResults, source)
			}
		}
	}
}
