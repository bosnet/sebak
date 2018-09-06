//
// Struct that bridges together components of a node
//
// NodeRunner bridges together the connection, storage and `LocalNode`.
// In this regard, it can be seen as a single node, and is used as such
// in unit tests.
//
package sebak

import (
	"errors"
	"sort"
	"time"

	logging "github.com/inconshreveable/log15"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/network"
	"boscoin.io/sebak/lib/node"
	"boscoin.io/sebak/lib/storage"
)

var DefaultHandleTransactionCheckerFuncs = []sebakcommon.CheckerFunc{
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
	INITBallotBroadcast,
}

var DefaultHandleSIGNBallotCheckerFuncs = []sebakcommon.CheckerFunc{
	BallotAlreadyVoted,
	BallotVote,
	BallotIsSameProposer,
	BallotCheckResult,
	SIGNBallotBroadcast,
}

var DefaultHandleACCEPTBallotCheckerFuncs = []sebakcommon.CheckerFunc{
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
	localNode          *sebaknode.LocalNode
	policy             sebakcommon.VotingThresholdPolicy
	network            sebaknetwork.Network
	consensus          *ISAAC
	connectionManager  *sebaknetwork.ConnectionManager
	storage            *sebakstorage.LevelDBBackend
	proposerCalculator ProposerCalculator

	handleTransactionCheckerFuncs  []sebakcommon.CheckerFunc
	handleBaseBallotCheckerFuncs   []sebakcommon.CheckerFunc
	handleINITBallotCheckerFuncs   []sebakcommon.CheckerFunc
	handleSIGNBallotCheckerFuncs   []sebakcommon.CheckerFunc
	handleACCEPTBallotCheckerFuncs []sebakcommon.CheckerFunc

	handleTransactionCheckerDeferFunc sebakcommon.CheckerDeferFunc
	handleBallotCheckerDeferFunc      sebakcommon.CheckerDeferFunc

	log logging.Logger

	timerExpireRound *time.Timer
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

	nr.SetProposerCalculator(SimpleProposerCalculator{})
	nr.policy.SetValidators(len(nr.localNode.GetValidators()) + 1) // including self

	nr.connectionManager = sebaknetwork.NewConnectionManager(
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

func (nr *NodeRunner) SetConf(conf *NodeRunnerConfiguration) {
}

func (nr *NodeRunner) Ready() {
	nodeHandler := NetworkHandlerNode{
		localNode: nr.localNode,
		network:   nr.network,
	}

	nr.network.AddHandler(sebaknetwork.UrlPathPrefixNode+"/", nodeHandler.NodeInfoHandler)
	nr.network.AddHandler(sebaknetwork.UrlPathPrefixNode+"/connect", nodeHandler.ConnectHandler)
	nr.network.AddHandler(sebaknetwork.UrlPathPrefixNode+"/message", nodeHandler.MessageHandler)
	nr.network.AddHandler(sebaknetwork.UrlPathPrefixNode+"/ballot", nodeHandler.BallotHandler)

	apiHandler := NetworkHandlerAPI{
		localNode: nr.localNode,
		network:   nr.network,
		storage:   nr.storage,
	}

	nr.network.AddHandler(
		sebaknetwork.UrlPathPrefixAPI+GetAccountHandlerPattern,
		apiHandler.GetAccountHandler,
	).Methods("GET")
	nr.network.AddHandler(
		sebaknetwork.UrlPathPrefixAPI+GetAccountTransactionsHandlerPattern,
		apiHandler.GetAccountTransactionsHandler,
	).Methods("GET")
	nr.network.AddHandler(
		sebaknetwork.UrlPathPrefixAPI+GetAccountOperationsHandlerPattern,
		apiHandler.GetAccountOperationsHandler,
	).Methods("GET")
	nr.network.AddHandler(
		sebaknetwork.UrlPathPrefixAPI+GetTransactionsHandlerPattern,
		apiHandler.GetTransactionsHandler,
	).Methods("GET")
	nr.network.AddHandler(
		sebaknetwork.UrlPathPrefixAPI+GetTransactionByHashHandlerPattern,
		apiHandler.GetTransactionByHashHandler,
	).Methods("GET")

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

func (nr *NodeRunner) SetHandleTransactionCheckerFuncs(
	deferFunc sebakcommon.CheckerDeferFunc,
	f ...sebakcommon.CheckerFunc,
) {
	if len(f) > 0 {
		nr.handleTransactionCheckerFuncs = f
	}

	if deferFunc == nil {
		deferFunc = sebakcommon.DefaultDeferFunc
	}

	nr.handleTransactionCheckerDeferFunc = deferFunc
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
	nr.handleTransactionCheckerDeferFunc = f
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
			if _, err := sebaknode.NewValidatorFromString(message.Data); err != nil {
				nr.log.Error("invalid validator data was received", "data", message.Data)
				continue
			}
		case sebaknetwork.TransactionMessage:
			if message.IsEmpty() {
				nr.log.Error("got empty transaction`")
			}
			err = nr.handleTransaction(message)
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

func (nr *NodeRunner) handleTransaction(message sebaknetwork.Message) (err error) {
	nr.log.Debug("got message`", "message", message.Head(50))

	checker := &MessageChecker{
		DefaultChecker: sebakcommon.DefaultChecker{Funcs: nr.handleTransactionCheckerFuncs},
		NodeRunner:     nr,
		LocalNode:      nr.localNode,
		NetworkID:      nr.networkID,
		Message:        message,
	}

	if err = sebakcommon.RunChecker(checker, nr.handleTransactionCheckerDeferFunc); err != nil {
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
	err = sebakcommon.RunChecker(baseChecker, nr.handleTransactionCheckerDeferFunc)
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
	err = sebakcommon.RunChecker(checker, nr.handleTransactionCheckerDeferFunc)
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
	nr.consensus.SetLatestRound(Round{})

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
	if nr.consensus.TransactionPool.Len() > MaxTransactionsInBallot {
		timeout = TimeoutProposeNewBallotFull
	} else {
		timeout = TimeoutProposeNewBallot
	}

	timer := time.NewTimer(timeout)
	go func() {
		<-timer.C

		if err := nr.proposeNewBallot(roundNumber); err != nil {
			nr.log.Error("failed to proposeNewBallot", "round", roundNumber, "error", err)
			go nr.StartNewRound(roundNumber)
		}
	}()

	return
}

func (nr *NodeRunner) proposeNewBallot(roundNumber uint64) error {
	// start new round
	round := Round{
		Number:      roundNumber,
		BlockHeight: nr.consensus.LatestConfirmedBlock.Height,
		BlockHash:   nr.consensus.LatestConfirmedBlock.Hash,
		TotalTxs:    nr.consensus.LatestConfirmedBlock.TotalTxs,
	}

	// collect incoming transactions from `TransactionPool`
	availableTransactions := nr.consensus.TransactionPool.AvailableTransactions()
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

	nr.ConnectionManager().Broadcast(ballot)

	runningRound, err := NewRunningRound(nr.localNode.Address(), *ballot)
	if err != nil {
		return err
	}
	rr := nr.consensus.RunningRounds
	rr[round.Hash()] = runningRound

	nr.Log().Debug("ballot broadcasted and voted", "runningRound", runningRound)

	return nil
}

func (nr *NodeRunner) CloseConsensus(ballot Ballot, confirmed bool) {
	nr.consensus.SetLatestRound(ballot.Round())

	if confirmed {
		go nr.StartNewRound(0)
	} else {
		go nr.StartNewRound(ballot.Round().Number + 1)
	}
}
