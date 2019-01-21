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

	connectionManager   network.ConnectionManager
	storage             *storage.LevelDBBackend
	proposerSelector    ProposerSelector
	log                 logging.Logger
	policy              voting.ThresholdPolicy
	syncer              SyncController
	latestReqSyncHeight uint64
	latestVotingBasis   voting.Basis

	LatestBallot  ballot.Ballot
	Node          *node.LocalNode
	RunningRounds map[ /* Round.Index() */ string]*RunningRound
	Conf          common.Config
}

// ISAAC should know network.ConnectionManager
// because the ISAAC uses connected validators when calculating proposer
func NewISAAC(node *node.LocalNode, p voting.ThresholdPolicy,
	cm network.ConnectionManager, st *storage.LevelDBBackend, conf common.Config, syncer SyncController) (is *ISAAC, err error) {

	is = &ISAAC{
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

// GenerateExpiredBallot create an expired ballot using voting.Basis and ballot.State.
// This function is used to create a ballot indicating
// that a node has expired to other nodes when a timeout occurs in the state.
func (is *ISAAC) GenerateExpiredBallot(basis voting.Basis, state ballot.State) (ballot.Ballot, error) {
	is.log.Debug("ISAAC.GenerateExpiredBallot", "basis", basis, "state", state)
	proposerAddr := is.SelectProposer(basis.Height, basis.Round)

	newExpiredBallot := ballot.NewBallot(is.Node.Address(), proposerAddr, basis, []string{})
	newExpiredBallot.SetVote(state, voting.EXP)

	config := is.Conf
	var err error

	opc, err := ballot.NewCollectTxFeeFromBallot(*newExpiredBallot, config.CommonAccountAddress)
	if err != nil {
		return ballot.Ballot{}, err
	}

	opi, err := ballot.NewInflationFromBallot(*newExpiredBallot, config.CommonAccountAddress, config.InitialBalance)
	if err != nil {
		return ballot.Ballot{}, err
	}

	ptx, err := ballot.NewProposerTransactionFromBallot(*newExpiredBallot, opc, opi)
	if err != nil {
		return ballot.Ballot{}, err
	}

	newExpiredBallot.SetProposerTransaction(ptx)
	newExpiredBallot.SignByProposer(is.Node.Keypair(), config.NetworkID)
	newExpiredBallot.Sign(is.Node.Keypair(), config.NetworkID)

	return *newExpiredBallot, nil
}

//
// Check if `basis` is a valid one for the current round of consensus
//
// Params:
//   basis = `voting.Basis` received for the current round of consensus
//   latestBlock = the latest block known to the node
//
// Returns:
//   bool = `true` if it's a valid voting basis, `false` otherwise
//
func (is *ISAAC) IsValidVotingBasis(basis voting.Basis, latestBlock block.Block) bool {
	is.RLock()
	defer is.RUnlock()

	lvb := is.LatestVotingBasis()

	log := is.log.New(logging.Ctx{
		"ballot-basis":        basis,
		"voting-basis":        lvb,
		"latest-block-height": latestBlock.Height,
		"latest-block-hash":   latestBlock.Hash,
	})

	if basis.Height != latestBlock.Height {
		log.Debug(
			"voting basis is invalid",
			"reason", "basis' height is different from latest block's height",
		)

		return false
	}
	if is.isInitRound(basis) {
		return true
	}

	// Note: If we have same height but different hash,
	// consensus is probably dead
	if basis.BlockHash != latestBlock.Hash {
		log.Error(
			"voting basis is invalid",
			"reason", "basis' block hash does not match latest block's hash",
		)
		return false
	}

	if basis.Height == lvb.Height {
		if basis.Round <= lvb.Round {
			log.Error(
				"voting basis is invalid",
				"reason", "basis' round is <= to latest voting basis' round",
			)
			return false
		}
	}
	return true
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

func (is *ISAAC) CanGetVotingResult(blt ballot.Ballot) (rv RoundVoteResult, vh voting.Hole, finished bool) {
	is.RLock()
	defer is.RUnlock()

	defer func() {
		is.log.Debug(
			"CanGetVotingResult",
			"ballot", blt.GetHash(),
			"round-vote-result", rv,
			"voting-hole", vh,
			"finished", finished,
		)
	}()

	rv = nil
	vh = voting.NOTYET
	finished = false

	runningRound, found := is.RunningRounds[blt.VotingBasis().Index()]
	if !found {
		// if RunningRound is not found, this ballot will be stopped.
		finished = true
		return
	}

	roundVote, err := runningRound.RoundVote(blt.Proposer())
	if err == nil {
		rv, vh, finished = roundVote.CanGetVotingResult(is.policy, blt.State(), is.log)
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
