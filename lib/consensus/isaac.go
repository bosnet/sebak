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

	latestBlock         block.Block
	connectionManager   network.ConnectionManager
	proposerSelector    ProposerSelector
	log                 logging.Logger
	policy              ballot.VotingThresholdPolicy
	nodesHeight         map[ /* Node.Address() */ string]uint64
	syncer              SyncController
	latestReqSyncHeight uint64

	LatestBallot  ballot.Ballot
	NetworkID     []byte
	Node          *node.LocalNode
	RunningRounds map[ /* Round.Index() */ string]*RunningRound
	LatestRound   round.Round
	Conf          common.Config
}

// ISAAC should know network.ConnectionManager
// because the ISAAC uses connected validators when calculating proposer
func NewISAAC(networkID []byte, node *node.LocalNode, p ballot.VotingThresholdPolicy,
	cm network.ConnectionManager, conf common.Config, syncer SyncController) (is *ISAAC, err error) {

	is = &ISAAC{
		NetworkID:         networkID,
		Node:              node,
		policy:            p,
		RunningRounds:     map[string]*RunningRound{},
		connectionManager: cm,
		proposerSelector:  SequentialSelector{cm},
		Conf:              conf,
		log:               log.New(logging.Ctx{"node": node.Alias()}),
		nodesHeight:       make(map[string]uint64),
		syncer:            syncer,
		LatestBallot:      ballot.Ballot{},
	}

	return
}

func (is *ISAAC) CloseConsensus(proposer string, round round.Round, vh ballot.VotingHole, transactionPool *transaction.Pool) (err error) {
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

	if vh == ballot.VotingYES {
		transactionPool.Remove(rr.Transactions[proposer]...)
	}

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

func (is *ISAAC) SaveNodeHeight(senderAddr string, height uint64) {
	is.nodesHeight[senderAddr] = height
}

func (is *ISAAC) IsAvailableRound(round round.Round) bool {
	if round.BlockHeight == is.latestBlock.Height {
		if is.isInitRound(round) {
			return true
		}

		if round.BlockHash != is.latestBlock.Hash {
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
	return is.LatestRound.BlockHash == "" && round.BlockHeight == common.GenesisBlockHeight
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
