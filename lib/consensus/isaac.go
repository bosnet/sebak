package consensus

import (
	"context"
	"sync"

	logging "github.com/inconshreveable/log15"

	"boscoin.io/sebak/lib/ballot"
	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/errors"
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
	syncer              SyncController
	latestReqSyncHeight uint64

	LatestBallot      ballot.Ballot
	NetworkID         []byte
	Node              *node.LocalNode
	RunningRounds     map[ /* Round.Index() */ string]*RunningRound
	latestVotingBasis voting.Basis
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
		syncer:            syncer,
		LatestBallot:      ballot.Ballot{},
	}

	return
}

func (is *ISAAC) SetLatestVotingBasis(basis voting.Basis) {
	is.Lock()
	defer is.Unlock()

	is.latestVotingBasis = basis
}

func (is *ISAAC) LatestVotingBasis() voting.Basis {
	is.RLock()
	defer is.RUnlock()

	return is.latestVotingBasis
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

		lvb := is.LatestVotingBasis()

		if basis.Height == lvb.Height {
			if basis.Round <= lvb.Round {
				return false
			}
		}
		return true
	}

	return false
}

func (is *ISAAC) isInitRound(basis voting.Basis) bool {
	return is.latestVotingBasis.BlockHash == "" && basis.Height == common.GenesisBlockHeight
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

func (is *ISAAC) CanGetVotingResult(b ballot.Ballot) (rv RoundVoteResult, vh voting.Hole, finished bool) {
	is.RLock()
	defer is.RUnlock()

	defer func() {
		is.log.Debug(
			"CanGetVotingResult",
			"ballot", b.GetHash(),
			"round-vote-result", rv,
			"voting-hole", vh,
			"finished", finished,
		)
	}()

	rv = nil
	vh = voting.NOTYET
	finished = false

	runningRound, found := is.RunningRounds[b.VotingBasis().Index()]
	if !found {
		// if RunningRound is not found, this ballot will be stopped.
		finished = true
		return
	}

	roundVote, err := runningRound.RoundVote(b.Proposer())
	if err == nil {
		rv, vh, finished = roundVote.CanGetVotingResult(is.policy, b.State(), is.log)
		return
	}

	return
}

func (is *ISAAC) IsVotedByNode(b ballot.Ballot, node string) (bool, error) {
	is.RLock()
	defer is.RUnlock()

	runningRound, found := is.RunningRounds[b.VotingBasis().Index()]
	if !found {
		return false, errors.RoundVoteNotFound
	}

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
