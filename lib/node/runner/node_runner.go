//
// Struct that bridges together components of a node
//
// NodeRunner bridges together the connection, storage and `LocalNode`.
// In this regard, it can be seen as a single node, and is used as such
// in unit tests.
//
package runner

import (
	"errors"
	"sort"
	"time"

	logging "github.com/inconshreveable/log15"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/consensus"
	"boscoin.io/sebak/lib/consensus/round"
	"boscoin.io/sebak/lib/network"
	"boscoin.io/sebak/lib/network/api"
	"boscoin.io/sebak/lib/node"
	"boscoin.io/sebak/lib/storage"
)

var DefaultHandleTransactionCheckerFuncs = []common.CheckerFunc{
	TransactionUnmarshal,
	HasTransaction,
	SaveTransactionHistory,
	MessageHasSameSource,
	MessageValidate,
	PushIntoTransactionPool,
	BroadcastTransaction,
}

var DefaultHandleBaseBallotCheckerFuncs = []common.CheckerFunc{
	BallotUnmarshal,
	BallotNotFromKnownValidators,
	BallotAlreadyFinished,
}

var DefaultHandleINITBallotCheckerFuncs = []common.CheckerFunc{
	BallotAlreadyVoted,
	BallotVote,
	BallotIsSameProposer,
	INITBallotValidateTransactions,
	INITBallotBroadcast,
}

var DefaultHandleSIGNBallotCheckerFuncs = []common.CheckerFunc{
	BallotAlreadyVoted,
	BallotVote,
	BallotIsSameProposer,
	BallotCheckResult,
	SIGNBallotBroadcast,
}

var DefaultHandleACCEPTBallotCheckerFuncs = []common.CheckerFunc{
	BallotAlreadyVoted,
	BallotVote,
	BallotIsSameProposer,
	BallotCheckResult,
	ACCEPTBallotStore,
}

type ProposerCalculator interface {
	Calculate(nr *NodeRunner, blockHeight uint64, roundNumber uint64) string
}

type NodeRunner struct {
	networkID          []byte
	localNode          *node.LocalNode
	policy             common.VotingThresholdPolicy
	network            network.Network
	consensus          *consensus.ISAAC
	connectionManager  *network.ConnectionManager
	storage            *storage.LevelDBBackend
	proposerCalculator ProposerCalculator

	handleTransactionCheckerFuncs  []common.CheckerFunc
	handleBaseBallotCheckerFuncs   []common.CheckerFunc
	handleINITBallotCheckerFuncs   []common.CheckerFunc
	handleSIGNBallotCheckerFuncs   []common.CheckerFunc
	handleACCEPTBallotCheckerFuncs []common.CheckerFunc

	handleTransactionCheckerDeferFunc common.CheckerDeferFunc
	handleBallotCheckerDeferFunc      common.CheckerDeferFunc

	log logging.Logger

	timerExpireRound *time.Timer
}

func NewNodeRunner(
	networkID string,
	localNode *node.LocalNode,
	policy common.VotingThresholdPolicy,
	n network.Network,
	c *consensus.ISAAC,
	storage *storage.LevelDBBackend,
) (nr *NodeRunner, err error) {
	nr = &NodeRunner{
		networkID: []byte(networkID),
		localNode: localNode,
		policy:    policy,
		network:   n,
		consensus: c,
		storage:   storage,
		log:       log.New(logging.Ctx{"node": localNode.Alias()}),
	}

	nr.SetProposerCalculator(SimpleProposerCalculator{})
	nr.policy.SetValidators(len(nr.localNode.GetValidators()) + 1) // including self

	nr.connectionManager = network.NewConnectionManager(
		nr.localNode,
		nr.network,
		nr.policy,
		nr.localNode.GetValidators(),
	)
	nr.network.AddWatcher(nr.connectionManager.ConnectionWatcher)

	nr.SetHandleTransactionCheckerFuncs(nil, DefaultHandleTransactionCheckerFuncs...)
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

func (nr *NodeRunner) SetConf(conf *consensus.ISAACConfiguration) {
}

func (nr *NodeRunner) Ready() {
	nodeHandler := NetworkHandlerNode{
		localNode: nr.localNode,
		network:   nr.network,
	}

	nr.network.AddHandler(network.UrlPathPrefixNode+"/", nodeHandler.NodeInfoHandler)
	nr.network.AddHandler(network.UrlPathPrefixNode+"/connect", nodeHandler.ConnectHandler)
	nr.network.AddHandler(network.UrlPathPrefixNode+"/message", nodeHandler.MessageHandler)
	nr.network.AddHandler(network.UrlPathPrefixNode+"/ballot", nodeHandler.BallotHandler)
	nr.network.AddHandler("/metrics", promhttp.Handler().ServeHTTP)

	apiHandler := api.NewNetworkHandlerAPI(
		nr.localNode,
		nr.network,
		nr.storage,
		network.UrlPathPrefixAPI,
	)

	nr.network.AddHandler(
		apiHandler.HandlerURLPattern(api.GetAccountHandlerPattern),
		apiHandler.GetAccountHandler,
	).Methods("GET")
	nr.network.AddHandler(
		apiHandler.HandlerURLPattern(api.GetAccountTransactionsHandlerPattern),
		apiHandler.GetTransactionsByAccountHandler,
	).Methods("GET")
	nr.network.AddHandler(
		apiHandler.HandlerURLPattern(api.GetAccountOperationsHandlerPattern),
		apiHandler.GetOperationsByAccountHandler,
	).Methods("GET")
	nr.network.AddHandler(
		apiHandler.HandlerURLPattern(api.GetTransactionsHandlerPattern),
		apiHandler.GetTransactionsHandler,
	).Methods("GET")
	nr.network.AddHandler(
		apiHandler.HandlerURLPattern(api.GetTransactionByHashHandlerPattern),
		apiHandler.GetTransactionByHashHandler,
	).Methods("GET")
	nr.network.AddHandler(
		apiHandler.HandlerURLPattern(api.GetTransactionOperationsHandlerPattern),
		apiHandler.GetOperationsByTxHashHandler,
	).Methods("GET")
	nr.network.AddHandler(
		apiHandler.HandlerURLPattern(api.PostTransactionPattern),
		nodeHandler.MessageHandler,
	).Methods("POST")

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
}

func (nr *NodeRunner) Node() *node.LocalNode {
	return nr.localNode
}

func (nr *NodeRunner) NetworkID() []byte {
	return nr.networkID
}

func (nr *NodeRunner) Network() network.Network {
	return nr.network
}

func (nr *NodeRunner) Consensus() *consensus.ISAAC {
	return nr.consensus
}

func (nr *NodeRunner) ConnectionManager() *network.ConnectionManager {
	return nr.connectionManager
}

func (nr *NodeRunner) Storage() *storage.LevelDBBackend {
	return nr.storage
}

func (nr *NodeRunner) Policy() common.VotingThresholdPolicy {
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

func (nr *NodeRunner) SetHandleTransactionCheckerFuncs(
	deferFunc common.CheckerDeferFunc,
	f ...common.CheckerFunc,
) {
	if len(f) > 0 {
		nr.handleTransactionCheckerFuncs = f
	}

	if deferFunc == nil {
		deferFunc = common.DefaultDeferFunc
	}

	nr.handleTransactionCheckerDeferFunc = deferFunc
}

func (nr *NodeRunner) SetHandleBaseBallotCheckerFuncs(f ...common.CheckerFunc) {
	nr.handleBaseBallotCheckerFuncs = f
}

func (nr *NodeRunner) SetHandleINITBallotCheckerFuncs(f ...common.CheckerFunc) {
	nr.handleINITBallotCheckerFuncs = f
}

func (nr *NodeRunner) SetHandleSIGNBallotCheckerFuncs(f ...common.CheckerFunc) {
	nr.handleSIGNBallotCheckerFuncs = f
}

func (nr *NodeRunner) SetHandleACCEPTBallotCheckerFuncs(f ...common.CheckerFunc) {
	nr.handleACCEPTBallotCheckerFuncs = f
}

func (nr *NodeRunner) SetHandleMessageCheckerDeferFunc(f common.CheckerDeferFunc) {
	nr.handleTransactionCheckerDeferFunc = f
}

func (nr *NodeRunner) handleMessage() {
	for message := range nr.network.ReceiveMessage() {
		var err error

		if message.IsEmpty() {
			nr.log.Error("got empty message")
			continue
		}
		switch message.Type {
		case common.ConnectMessage:
			if _, err := node.NewValidatorFromString(message.Data); err != nil {
				nr.log.Error("invalid validator data was received", "data", message.Data, "error", err)
				continue
			}
		case common.TransactionMessage:
			err = nr.handleTransaction(message)
		case common.BallotMessage:
			err = nr.handleBallotMessage(message)
		default:
			err = errors.New("got unknown message")
		}

		if err != nil {
			if _, ok := err.(common.CheckerStop); ok {
				continue
			}
			nr.log.Error("failed to handle message", "message", message.Head(50), "error", err)
		}
	}
}

func (nr *NodeRunner) handleTransaction(message common.NetworkMessage) (err error) {
	nr.log.Debug("got transaction", "transaction", message.Head(50))

	checker := &MessageChecker{
		DefaultChecker: common.DefaultChecker{Funcs: nr.handleTransactionCheckerFuncs},
		NodeRunner:     nr,
		LocalNode:      nr.localNode,
		NetworkID:      nr.networkID,
		Message:        message,
	}

	if err = common.RunChecker(checker, nr.handleTransactionCheckerDeferFunc); err != nil {
		if _, ok := err.(common.CheckerErrorStop); !ok {
			nr.log.Error("failed to handle transaction", "error", err)
		}
		return
	}

	return
}

func (nr *NodeRunner) handleBallotMessage(message common.NetworkMessage) (err error) {
	nr.log.Debug("got ballot", "message", message.Head(50))

	baseChecker := &BallotChecker{
		DefaultChecker: common.DefaultChecker{Funcs: nr.handleBaseBallotCheckerFuncs},
		NodeRunner:     nr,
		LocalNode:      nr.localNode,
		NetworkID:      nr.networkID,
		Message:        message,
		Log:            nr.Log(),
		VotingHole:     common.VotingNOTYET,
	}
	err = common.RunChecker(baseChecker, nr.handleTransactionCheckerDeferFunc)
	if err != nil {
		if _, ok := err.(common.CheckerErrorStop); !ok {
			nr.log.Error("failed to handle ballot", "error", err, "state", "")
			return
		}
	}

	var checkerFuncs []common.CheckerFunc
	switch baseChecker.Ballot.State() {
	case common.BallotStateINIT:
		checkerFuncs = DefaultHandleINITBallotCheckerFuncs
	case common.BallotStateSIGN:
		checkerFuncs = DefaultHandleSIGNBallotCheckerFuncs
	case common.BallotStateACCEPT:
		checkerFuncs = DefaultHandleACCEPTBallotCheckerFuncs
	}

	checker := &BallotChecker{
		DefaultChecker: common.DefaultChecker{Funcs: checkerFuncs},
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
	err = common.RunChecker(checker, nr.handleTransactionCheckerDeferFunc)
	if err != nil {
		if stopped, ok := err.(common.CheckerStop); ok {
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
	var latestBlock block.Block
	if latestBlock, err = block.GetLatestBlock(nr.storage); err != nil {
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
			if _, found := common.InStringArray(connected, address); !found {
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
		"caught up network and connected to all validators",
		"connected", nr.Policy().Connected(),
		"validators", nr.Policy().Validators(),
	)

	go nr.startRound()
}

func (nr *NodeRunner) startRound() {
	// check whether current running rounds exist
	if len(nr.consensus.RunningRounds) > 0 {
		return
	}

	nr.StartNewRound(0)
}

func (nr *NodeRunner) CalculateProposer(blockHeight uint64, roundNumber uint64) string {
	return nr.proposerCalculator.Calculate(nr, blockHeight, roundNumber)
}

func (nr *NodeRunner) StartNewRound(roundNumber uint64) {
	if nr.timerExpireRound != nil {
		if !nr.timerExpireRound.Stop() {
			<-nr.timerExpireRound.C
		}
	}

	// wait for new ballot from new proposer
	nr.timerExpireRound = time.AfterFunc(TimeoutExpireRound,
		func() {
			nr.StartNewRound(roundNumber + 1)
		})

	proposer := nr.CalculateProposer(
		nr.consensus.LatestConfirmedBlock.Height,
		roundNumber,
	)

	log.Debug("calculated proposer", "proposer", proposer)

	if proposer != nr.localNode.Address() {
		return
	}

	nr.readyToProposeNewBallot(roundNumber)

	return
}

func (nr *NodeRunner) readyToProposeNewBallot(roundNumber uint64) {
	var timeout time.Duration
	// if incoming transaactions are over `MaxTransactionsInBallot`, just
	// start.
	if nr.consensus.TransactionPool.Len() > common.MaxTransactionsInBallot {
		timeout = TimeoutProposeNewBallotFull
	} else {
		timeout = TimeoutProposeNewBallot
	}

	time.AfterFunc(timeout,
		func() {
			if err := nr.proposeNewBallot(roundNumber); err != nil {
				nr.log.Error("failed to proposeNewBallot", "round", roundNumber, "error", err)
				nr.StartNewRound(roundNumber)
			}
		})

	return
}

func (nr *NodeRunner) proposeNewBallot(roundNumber uint64) error {
	round := round.Round{
		Number:      roundNumber,
		BlockHeight: nr.consensus.LatestConfirmedBlock.Height,
		BlockHash:   nr.consensus.LatestConfirmedBlock.Hash,
		TotalTxs:    nr.consensus.LatestConfirmedBlock.TotalTxs,
	}

	// collect incoming transactions from `TransactionPool`
	availableTransactions := nr.consensus.TransactionPool.AvailableTransactions(common.MaxTransactionsInBallot)
	nr.log.Debug("new round proposed", "round", round, "transactions", availableTransactions)

	transactionsChecker := &BallotTransactionChecker{
		DefaultChecker: common.DefaultChecker{Funcs: handleBallotTransactionCheckerFuncs},
		NodeRunner:     nr,
		LocalNode:      nr.localNode,
		NetworkID:      nr.networkID,
		Transactions:   availableTransactions,
		CheckAll:       true,
		VotingHole:     common.VotingNOTYET,
	}

	if err := common.RunChecker(transactionsChecker, common.DefaultDeferFunc); err != nil {
		if _, ok := err.(common.CheckerErrorStop); !ok {
			nr.log.Error("error occurred in BallotTransactionChecker", "error", err)
		}
	}

	// remove invalid transactions
	nr.Consensus().TransactionPool.Remove(transactionsChecker.InvalidTransactions()...)

	ballot := block.NewBallot(nr.localNode, round, transactionsChecker.ValidTransactions)
	ballot.SetVote(common.BallotStateINIT, common.VotingYES)
	ballot.Sign(nr.localNode.Keypair(), nr.networkID)

	nr.log.Debug("new ballot created", "ballot", ballot)

	nr.ConnectionManager().Broadcast(ballot)

	runningRound, err := consensus.NewRunningRound(nr.localNode.Address(), *ballot)
	if err != nil {
		return err
	}
	rr := nr.consensus.RunningRounds
	rr[round.Hash()] = runningRound

	nr.log.Debug("ballot broadcasted and voted", "runningRound", runningRound)

	return nil
}

func (nr *NodeRunner) CloseConsensus(ballot block.Ballot, confirmed bool) {
	nr.consensus.SetLatestRound(ballot.Round())

	if confirmed {
		go nr.StartNewRound(0)
	} else {
		go nr.StartNewRound(ballot.Round().Number + 1)
	}
}
