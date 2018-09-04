//
// Struct that bridges together components of a node
//
// NodeRunner bridges together the connection, storage and `LocalNode`.
// In this regard, it can be seen as a single node, and is used as such
// in unit tests.
//
package sebak

import (
	"context"
	"errors"
	"sort"
	"time"

	logging "github.com/inconshreveable/log15"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/network"
	"boscoin.io/sebak/lib/node"
	"boscoin.io/sebak/lib/round"
	"boscoin.io/sebak/lib/storage"
)

var DefaultHandleMessageFromClientCheckerFuncs = []sebakcommon.CheckerFunc{
	TransactionUnmarshal,
	HasTransaction,
	SaveTransactionHistory,
	MessageHasSameSource,
	MessageValidate,
	PushIntoTransactionPool,
	BroadcastTransaction,
}

var DefaultHandleBaseBallotCheckerFuncs = []sebakcommon.CheckerFunc{
	BallotUnmarshal,
	BallotNotFromKnownValidators,
	BallotAlreadyFinished,
}

var DefaultHandleINITBallotCheckerFuncs = []sebakcommon.CheckerFunc{
	BallotAlreadyVoted,
	BallotVote,
	BallotIsSameProposer,
	INITBallotValidateTransactions,
	SIGNBallotBroadcast,
	TransitStateToSIGN,
}

var DefaultHandleSIGNBallotCheckerFuncs = []sebakcommon.CheckerFunc{
	BallotAlreadyVoted,
	BallotVote,
	BallotIsSameProposer,
	BallotCheckResult,
	ACCEPTBallotBroadcast,
	TransitStateToACCEPT,
}

var DefaultHandleACCEPTBallotCheckerFuncs = []sebakcommon.CheckerFunc{
	BallotAlreadyVoted,
	BallotVote,
	BallotIsSameProposer,
	BallotCheckResult,
	FinishedBallotStore,
}

type ProposerCalculator interface {
	Calculate(nr *NodeRunner, blockHeight uint64, roundNumber uint64) string
}

type NodeRunner struct {
	networkID          []byte
	localNode          *sebaknode.LocalNode
	policy             sebakcommon.VotingThresholdPolicy
	network            sebaknetwork.Network
	consensus          *ISAAC
	connectionManager  *sebaknetwork.ConnectionManager
	storage            *sebakstorage.LevelDBBackend
	proposerCalculator ProposerCalculator
	isaacStateManager  *IsaacStateManager

	handleMessageFromClientCheckerFuncs []sebakcommon.CheckerFunc
	handleBaseBallotCheckerFuncs        []sebakcommon.CheckerFunc
	handleINITBallotCheckerFuncs        []sebakcommon.CheckerFunc
	handleSIGNBallotCheckerFuncs        []sebakcommon.CheckerFunc
	handleACCEPTBallotCheckerFuncs      []sebakcommon.CheckerFunc

	handleMessageCheckerDeferFunc sebakcommon.CheckerDeferFunc

	ctx context.Context
	log logging.Logger
}

func NewNodeRunner(
	networkID string,
	localNode *sebaknode.LocalNode,
	policy sebakcommon.VotingThresholdPolicy,
	network sebaknetwork.Network,
	consensus *ISAAC,
	storage *sebakstorage.LevelDBBackend,
) (nr *NodeRunner, err error) {
	nr = &NodeRunner{
		networkID: []byte(networkID),
		localNode: localNode,
		policy:    policy,
		network:   network,
		consensus: consensus,
		storage:   storage,
		log:       log.New(logging.Ctx{"node": localNode.Alias()}),
	}
	nr.isaacStateManager = NewIsaacStateManager(nr)
	nr.ctx = context.WithValue(context.Background(), "localNode", localNode)
	nr.ctx = context.WithValue(nr.ctx, "networkID", nr.networkID)
	nr.ctx = context.WithValue(nr.ctx, "storage", nr.storage)

	nr.SetProposerCalculator(SimpleProposerCalculator{})
	nr.policy.SetValidators(len(nr.localNode.GetValidators()) + 1) // including self

	nr.connectionManager = sebaknetwork.NewConnectionManager(
		nr.localNode,
		nr.network,
		nr.policy,
		nr.localNode.GetValidators(),
	)

	nr.connectionManager.SetBroadcastor(sebaknetwork.SimpleBroadcastor{})
	nr.network.AddWatcher(nr.connectionManager.ConnectionWatcher)

	nr.SetHandleMessageFromClientCheckerFuncs(DefaultHandleMessageFromClientCheckerFuncs...)
	nr.SetHandleBaseBallotCheckerFuncs(DefaultHandleBaseBallotCheckerFuncs...)
	nr.SetHandleINITBallotCheckerFuncs(DefaultHandleINITBallotCheckerFuncs...)
	nr.SetHandleSIGNBallotCheckerFuncs(DefaultHandleSIGNBallotCheckerFuncs...)
	nr.SetHandleACCEPTBallotCheckerFuncs(DefaultHandleACCEPTBallotCheckerFuncs...)

	return
}

type SimpleProposerCalculator struct {
}

func (c SimpleProposerCalculator) Calculate(nr *NodeRunner, blockHeight uint64, roundNumber uint64) string {
	candidates := sort.StringSlice(nr.connectionManager.AllValidators())
	candidates.Sort()

	return candidates[(blockHeight+roundNumber)%uint64(len(candidates))]
}

func (nr *NodeRunner) SetProposerCalculator(c ProposerCalculator) {
	nr.proposerCalculator = c
}

func (nr *NodeRunner) SetConf(conf *IsaacConfiguration) {
	nr.isaacStateManager.SetConf(conf)
}

func (nr *NodeRunner) SetBroadcastor(b sebaknetwork.Broadcastor) {
	nr.connectionManager.SetBroadcastor(b)
}

func (nr *NodeRunner) Ready() {
	nr.network.SetContext(nr.ctx)
	nr.network.AddHandler(nr.ctx, AddAPIHandlers(nr.storage))
	nr.network.Ready()
}

func (nr *NodeRunner) Start() (err error) {
	nr.log.Debug("NodeRunner started")
	nr.Ready()

	go nr.handleMessage()
	go nr.ConnectValidators()
	go nr.InitRound()

	if err = nr.network.Start(); err != nil {
		return
	}

	return
}

func (nr *NodeRunner) Stop() {
	nr.network.Stop()
	nr.isaacStateManager.Stop()
}

func (nr *NodeRunner) Node() *sebaknode.LocalNode {
	return nr.localNode
}

func (nr *NodeRunner) NetworkID() []byte {
	return nr.networkID
}

func (nr *NodeRunner) Network() sebaknetwork.Network {
	return nr.network
}

func (nr *NodeRunner) Consensus() *ISAAC {
	return nr.consensus
}

func (nr *NodeRunner) ConnectionManager() *sebaknetwork.ConnectionManager {
	return nr.connectionManager
}

func (nr *NodeRunner) Storage() *sebakstorage.LevelDBBackend {
	return nr.storage
}

func (nr *NodeRunner) Policy() sebakcommon.VotingThresholdPolicy {
	return nr.policy
}

func (nr *NodeRunner) Log() logging.Logger {
	return nr.log
}

func (nr *NodeRunner) ConnectValidators() {
	ticker := time.NewTicker(time.Millisecond * 5)
	for _ = range ticker.C {
		if !nr.network.IsReady() {
			continue
		}

		ticker.Stop()
		break
	}
	nr.log.Debug("current node is ready")
	nr.log.Debug("trying to connect to the validators", "validators", nr.localNode.GetValidators())

	nr.log.Debug("initializing connectionManager for validators")
	nr.connectionManager.Start()
}

func (nr *NodeRunner) SetHandleMessageFromClientCheckerFuncs(f ...sebakcommon.CheckerFunc) {
	nr.handleMessageFromClientCheckerFuncs = f
}

func (nr *NodeRunner) SetHandleBaseBallotCheckerFuncs(f ...sebakcommon.CheckerFunc) {
	nr.handleBaseBallotCheckerFuncs = f
}

func (nr *NodeRunner) SetHandleINITBallotCheckerFuncs(f ...sebakcommon.CheckerFunc) {
	nr.handleINITBallotCheckerFuncs = f
}

func (nr *NodeRunner) SetHandleSIGNBallotCheckerFuncs(f ...sebakcommon.CheckerFunc) {
	nr.handleSIGNBallotCheckerFuncs = f
}

func (nr *NodeRunner) SetHandleACCEPTBallotCheckerFuncs(f ...sebakcommon.CheckerFunc) {
	nr.handleACCEPTBallotCheckerFuncs = f
}

func (nr *NodeRunner) SetHandleMessageCheckerDeferFunc(f sebakcommon.CheckerDeferFunc) {
	nr.handleMessageCheckerDeferFunc = f
}

func (nr *NodeRunner) handleMessage() {
	for message := range nr.network.ReceiveMessage() {
		var err error

		if message.IsEmpty() {
			nr.log.Error("got empty message`")
			continue
		}
		switch message.Type {
		case sebaknetwork.ConnectMessage:
			//nr.log.Debug("got connect", "message", message.Head(50))
			if _, err = sebaknode.NewValidatorFromString(message.Data); err != nil {
				err = errors.New("invalid validator data was received")
			}
		case sebaknetwork.MessageFromClient:
			err = nr.handleMessageFromClient(message)
		case sebaknetwork.BallotMessage:
			err = nr.handleBallotMessage(message)
		default:
			err = errors.New("got unknown message")
		}

		if err != nil {
			if _, ok := err.(sebakcommon.CheckerStop); ok {
				continue
			}
			nr.log.Error("failed to handle sebaknetwork.Message", "message", message.Head(50), "error", err)
		}
	}
}

func (nr *NodeRunner) handleMessageFromClient(message sebaknetwork.Message) (err error) {
	nr.log.Debug("got message`", "message", message.Head(50))

	checker := &MessageChecker{
		DefaultChecker: sebakcommon.DefaultChecker{Funcs: nr.handleMessageFromClientCheckerFuncs},
		NodeRunner:     nr,
		LocalNode:      nr.localNode,
		NetworkID:      nr.networkID,
		Message:        message,
	}

	if err = sebakcommon.RunChecker(checker, nr.handleMessageCheckerDeferFunc); err != nil {
		if _, ok := err.(sebakcommon.CheckerErrorStop); !ok {
			nr.log.Error("failed to handle message from client", "error", err)
		}
		return
	}

	return
}

func (nr *NodeRunner) handleBallotMessage(message sebaknetwork.Message) (err error) {
	nr.log.Debug("got ballot", "message", message.Head(50))

	baseChecker := &BallotChecker{
		DefaultChecker: sebakcommon.DefaultChecker{Funcs: nr.handleBaseBallotCheckerFuncs},
		NodeRunner:     nr,
		LocalNode:      nr.localNode,
		NetworkID:      nr.networkID,
		Message:        message,
		Log:            nr.Log(),
		VotingHole:     sebakcommon.VotingNOTYET,
	}
	err = sebakcommon.RunChecker(baseChecker, nr.handleMessageCheckerDeferFunc)
	if err != nil {
		if _, ok := err.(sebakcommon.CheckerErrorStop); !ok {
			nr.log.Error("failed to handle ballot", "error", err, "state", "base")
			return
		}
	}

	var checkerFuncs []sebakcommon.CheckerFunc
	switch baseChecker.Ballot.State() {
	case sebakcommon.BallotStateINIT:
		checkerFuncs = DefaultHandleINITBallotCheckerFuncs
	case sebakcommon.BallotStateSIGN:
		checkerFuncs = DefaultHandleSIGNBallotCheckerFuncs
	case sebakcommon.BallotStateACCEPT:
		checkerFuncs = DefaultHandleACCEPTBallotCheckerFuncs
	}

	checker := &BallotChecker{
		DefaultChecker: sebakcommon.DefaultChecker{Funcs: checkerFuncs},
		NodeRunner:     nr,
		LocalNode:      nr.localNode,
		NetworkID:      nr.networkID,
		Message:        message,
		Ballot:         baseChecker.Ballot,
		VotingHole:     baseChecker.VotingHole,
		IsNew:          baseChecker.IsNew,
		RoundVote:      baseChecker.RoundVote,
		Log:            baseChecker.Log,
	}
	err = sebakcommon.RunChecker(checker, nr.handleMessageCheckerDeferFunc)
	if err != nil {
		if stopped, ok := err.(sebakcommon.CheckerStop); ok {
			nr.log.Debug(
				"stopped to handle ballot",
				"state", baseChecker.Ballot.State(),
				"reason", stopped.Error(),
			)
		} else {
			nr.log.Error("failed to handle ballot", "error", err, "state", baseChecker.Ballot.State())
			return
		}
	}

	return
}

func (nr *NodeRunner) InitRound() {
	// get latest blocks
	var err error
	var latestBlock Block
	if latestBlock, err = GetLatestBlock(nr.storage); err != nil {
		panic(err)
	}

	nr.consensus.SetLatestConsensusedBlock(latestBlock)
	nr.consensus.SetLatestRound(round.Round{})

	ticker := time.NewTicker(time.Millisecond * 5)
	for _ = range ticker.C {
		var notFound bool
		connected := nr.connectionManager.AllConnected()
		if len(connected) < 1 {
			continue
		}

		for address, _ := range nr.localNode.GetValidators() {
			if _, found := sebakcommon.InStringArray(connected, address); !found {
				notFound = true
				break
			}
		}
		if !notFound {
			ticker.Stop()
			break
		}
	}

	nr.log.Debug(
		"caught up with network and connected to all validators",
		"connected", nr.Policy().Connected(),
		"validators", nr.Policy().Validators(),
	)

	nr.StartStateManager()
}

func (nr *NodeRunner) StartStateManager() {
	// check whether current running rounds exist
	if len(nr.consensus.RunningRounds) > 0 {
		return
	}

	go nr.isaacStateManager.Start()
	nr.isaacStateManager.NextHeight()
	return
}

func (nr *NodeRunner) CalculateProposer(blockHeight uint64, roundNumber uint64) string {
	return nr.proposerCalculator.Calculate(nr, blockHeight, roundNumber)
}

func (nr *NodeRunner) TransitIsaacState(round round.Round, ballotState sebakcommon.BallotState) {
	nr.isaacStateManager.TransitIsaacState(round, ballotState)
}

func (nr *NodeRunner) proposeNewBallot(roundNumber uint64) error {
	round := round.Round{
		Number:      roundNumber,
		BlockHeight: nr.consensus.LatestConfirmedBlock.Height,
		BlockHash:   nr.consensus.LatestConfirmedBlock.Hash,
		TotalTxs:    nr.consensus.LatestConfirmedBlock.TotalTxs,
	}

	// collect incoming transactions from `TransactionPool`
	availableTransactions := nr.consensus.TransactionPool.AvailableTransactions(nr.isaacStateManager.conf)
	nr.log.Debug("new round proposed", "round", round, "transactions", availableTransactions)

	transactionsChecker := &BallotTransactionChecker{
		DefaultChecker: sebakcommon.DefaultChecker{Funcs: handleBallotTransactionCheckerFuncs},
		NodeRunner:     nr,
		LocalNode:      nr.localNode,
		NetworkID:      nr.networkID,
		Transactions:   availableTransactions,
		CheckAll:       true,
		VotingHole:     sebakcommon.VotingNOTYET,
	}

	if err := sebakcommon.RunChecker(transactionsChecker, sebakcommon.DefaultDeferFunc); err != nil {
		if _, ok := err.(sebakcommon.CheckerErrorStop); !ok {
			nr.log.Error("error occurred in BallotTransactionChecker", "error", err)
		}
	}

	// remove invalid transactions
	nr.Consensus().TransactionPool.Remove(transactionsChecker.InvalidTransactions()...)

	ballot := NewBallot(nr.localNode, round, transactionsChecker.ValidTransactions)
	ballot.SetVote(sebakcommon.BallotStateINIT, sebakcommon.VotingYES)
	ballot.Sign(nr.localNode.Keypair(), nr.networkID)

	nr.log.Debug("new ballot created", "ballot", ballot)

	nr.ConnectionManager().Broadcast(*ballot)

	runningRound, err := NewRunningRound(nr.localNode.Address(), *ballot)
	if err != nil {
		return err
	}
	rr := nr.consensus.RunningRounds
	rr[round.Hash()] = runningRound

	nr.Log().Debug("ballot broadcasted and voted", "runningRound", runningRound)

	return nil
}

func (nr *NodeRunner) CloseConsensus(round round.Round) {
	nr.consensus.SetLatestRound(round)
	nr.isaacStateManager.NextHeight()
}
