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
	"time"

	logging "github.com/inconshreveable/log15"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"boscoin.io/sebak/lib/ballot"
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
	SIGNBallotBroadcast,
	TransitStateToSIGN,
}

var DefaultHandleSIGNBallotCheckerFuncs = []common.CheckerFunc{
	BallotAlreadyVoted,
	BallotVote,
	BallotIsSameProposer,
	BallotCheckResult,
	ACCEPTBallotBroadcast,
	TransitStateToACCEPT,
}

var DefaultHandleACCEPTBallotCheckerFuncs = []common.CheckerFunc{
	BallotAlreadyVoted,
	BallotVote,
	BallotIsSameProposer,
	BallotCheckResult,
	FinishedBallotStore,
}

type NodeRunner struct {
	networkID         []byte
	localNode         *node.LocalNode
	policy            ballot.VotingThresholdPolicy
	network           network.Network
	consensus         *consensus.ISAAC
	connectionManager *network.ConnectionManager
	storage           *storage.LevelDBBackend
	isaacStateManager *ISAACStateManager

	handleTransactionCheckerFuncs  []common.CheckerFunc
	handleBaseBallotCheckerFuncs   []common.CheckerFunc
	handleINITBallotCheckerFuncs   []common.CheckerFunc
	handleSIGNBallotCheckerFuncs   []common.CheckerFunc
	handleACCEPTBallotCheckerFuncs []common.CheckerFunc

	handleTransactionCheckerDeferFunc common.CheckerDeferFunc
	handleBallotCheckerDeferFunc      common.CheckerDeferFunc

	log logging.Logger
}

func NewNodeRunner(
	networkID string,
	localNode *node.LocalNode,
	policy ballot.VotingThresholdPolicy,
	n network.Network,
	consensus *consensus.ISAAC,
	storage *storage.LevelDBBackend,
	conf *consensus.ISAACConfiguration,
) (nr *NodeRunner, err error) {
	nr = &NodeRunner{
		networkID: []byte(networkID),
		localNode: localNode,
		policy:    policy,
		network:   n,
		consensus: consensus,
		storage:   storage,
		log:       log.New(logging.Ctx{"node": localNode.Alias()}),
	}
	nr.isaacStateManager = NewISAACStateManager(nr, conf)

	nr.policy.SetValidators(len(nr.localNode.GetValidators()) + 1) // including self

	nr.connectionManager = consensus.ConnectionManager()
	nr.network.AddWatcher(nr.connectionManager.ConnectionWatcher)

	nr.SetHandleTransactionCheckerFuncs(nil, DefaultHandleTransactionCheckerFuncs...)
	nr.SetHandleBaseBallotCheckerFuncs(DefaultHandleBaseBallotCheckerFuncs...)
	nr.SetHandleINITBallotCheckerFuncs(DefaultHandleINITBallotCheckerFuncs...)
	nr.SetHandleSIGNBallotCheckerFuncs(DefaultHandleSIGNBallotCheckerFuncs...)
	nr.SetHandleACCEPTBallotCheckerFuncs(DefaultHandleACCEPTBallotCheckerFuncs...)

	return
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
	nr.isaacStateManager.Stop()
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

func (nr *NodeRunner) Policy() ballot.VotingThresholdPolicy {
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
		VotingHole:     ballot.VotingNOTYET,
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
	case ballot.StateINIT:
		checkerFuncs = DefaultHandleINITBallotCheckerFuncs
	case ballot.StateSIGN:
		checkerFuncs = DefaultHandleSIGNBallotCheckerFuncs
	case ballot.StateACCEPT:
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

	nr.StartStateManager()
}

func (nr *NodeRunner) StartStateManager() {
	// check whether current running rounds exist
	if len(nr.consensus.RunningRounds) > 0 {
		return
	}

	nr.isaacStateManager.Start()
	nr.isaacStateManager.NextHeight()
	return
}

func (nr *NodeRunner) StopStateManager() {
	// check whether current running rounds exist
	nr.isaacStateManager.Stop()
	return
}

func (nr *NodeRunner) TransitISAACState(round round.Round, ballotState ballot.State) {
	nr.isaacStateManager.TransitISAACState(round, ballotState)
}

func (nr *NodeRunner) proposeNewBallot(roundNumber uint64) error {
	b := nr.consensus.LatestConfirmedBlock()
	round := round.Round{
		Number:      roundNumber,
		BlockHeight: b.Height,
		BlockHash:   b.Hash,
		TotalTxs:    b.TotalTxs,
	}

	// collect incoming transactions from `TransactionPool`
	availableTransactions := nr.consensus.TransactionPool.AvailableTransactions(int(nr.isaacStateManager.Conf.TransactionsLimit))
	nr.log.Debug("new round proposed", "round", round, "transactions", availableTransactions)

	transactionsChecker := &BallotTransactionChecker{
		DefaultChecker: common.DefaultChecker{Funcs: handleBallotTransactionCheckerFuncs},
		NodeRunner:     nr,
		LocalNode:      nr.localNode,
		NetworkID:      nr.networkID,
		Transactions:   availableTransactions,
		CheckAll:       true,
		VotingHole:     ballot.VotingNOTYET,
	}

	if err := common.RunChecker(transactionsChecker, common.DefaultDeferFunc); err != nil {
		if _, ok := err.(common.CheckerErrorStop); !ok {
			nr.log.Error("error occurred in BallotTransactionChecker", "error", err)
		}
	}

	// remove invalid transactions
	nr.Consensus().TransactionPool.Remove(transactionsChecker.InvalidTransactions()...)

	theBallot := block.NewBallot(nr.localNode, round, transactionsChecker.ValidTransactions)
	theBallot.SetVote(ballot.StateINIT, ballot.VotingYES)
	theBallot.Sign(nr.localNode.Keypair(), nr.networkID)

	nr.log.Debug("new ballot created", "ballot", theBallot)

	nr.ConnectionManager().Broadcast(*theBallot)

	runningRound, err := consensus.NewRunningRound(nr.localNode.Address(), *theBallot)
	if err != nil {
		return err
	}

	nr.consensus.AddRunningRound(round.Hash(), runningRound)

	nr.log.Debug("ballot broadcasted and voted", "runningRound", runningRound)

	return nil
}
