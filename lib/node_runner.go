//
// Struct that bridges together components of a node
//
// NodeRunner bridges together the connection, storage and `LocalNode`.
// In this regard, it can be seen as a single node, and is used as such
// in unit tests.
//
package sebak

import (
	"boscoin.io/sebak/lib/network"
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	logging "github.com/inconshreveable/log15"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/node"
	"boscoin.io/sebak/lib/storage"
)

var (
	TimeoutExpireRound          time.Duration = time.Second * 10
	TimeoutProposeNewBallot     time.Duration = time.Second * 2
	TimeoutProposeNewBallotFull time.Duration = time.Second * 1
	MaxTransactionsInBallot     int           = 1000
)

var DefaultHandleMessageFromClientCheckerFuncs = []sebakcommon.CheckerFunc{
	CheckNodeRunnerHandleMessageTransactionUnmarshal,
	CheckNodeRunnerHandleMessageHasTransactionAlready,
	CheckNodeRunnerHandleMessageHistory,
	CheckNodeRunnerHandleMessagePushIntoTransactionPool,
	CheckNodeRunnerHandleMessageTransactionBroadcast,
}

var DefaultHandleBaseBallotCheckerFuncs = []sebakcommon.CheckerFunc{
	CheckNodeRunnerHandleBallotUnmarshal,
	CheckNodeRunnerHandleBallotNotFromKnownValidators,
	CheckNodeRunnerHandleBallotAlreadyFinished,
}

var DefaultHandleINITBallotCheckerFuncs = []sebakcommon.CheckerFunc{
	CheckNodeRunnerHandleBallotAlreadyVoted,
	CheckNodeRunnerHandleBallotVote,
	CheckNodeRunnerHandleBallotIsSameProposer,
	CheckNodeRunnerHandleINITBallotValidateTransactions,
	CheckNodeRunnerHandleINITBallotBroadcast,
}

var DefaultHandleSIGNBallotCheckerFuncs = []sebakcommon.CheckerFunc{
	CheckNodeRunnerHandleBallotAlreadyVoted,
	CheckNodeRunnerHandleBallotVote,
	CheckNodeRunnerHandleBallotIsSameProposer,
	CheckNodeRunnerHandleBallotCheckResult,
	CheckNodeRunnerHandleSIGNBallotBroadcast,
}

var DefaultHandleACCEPTBallotCheckerFuncs = []sebakcommon.CheckerFunc{
	CheckNodeRunnerHandleBallotAlreadyVoted,
	CheckNodeRunnerHandleBallotVote,
	CheckNodeRunnerHandleBallotIsSameProposer,
	CheckNodeRunnerHandleBallotCheckResult,
	CheckNodeRunnerHandleACCEPTBallotStore,
}

type NodeRunner struct {
	networkID         []byte
	localNode         *sebaknode.LocalNode
	policy            sebakcommon.VotingThresholdPolicy
	network           sebaknetwork.Network
	consensus         *ISAAC
	connectionManager *sebaknetwork.ConnectionManager
	storage           *sebakstorage.LevelDBBackend

	handleMessageFromClientCheckerFuncs []sebakcommon.CheckerFunc
	handleBaseBallotCheckerFuncs        []sebakcommon.CheckerFunc
	handleINITBallotCheckerFuncs        []sebakcommon.CheckerFunc
	handleSIGNBallotCheckerFuncs        []sebakcommon.CheckerFunc
	handleACCEPTBallotCheckerFuncs      []sebakcommon.CheckerFunc

	handleMessageCheckerDeferFunc sebakcommon.CheckerDeferFunc

	ctx context.Context
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
	nr.ctx = context.WithValue(context.Background(), "localNode", localNode)
	nr.ctx = context.WithValue(nr.ctx, "networkID", nr.networkID)
	nr.ctx = context.WithValue(nr.ctx, "storage", nr.storage)

	nr.connectionManager = sebaknetwork.NewConnectionManager(
		nr.localNode,
		nr.network,
		nr.policy,
		nr.localNode.GetValidators(),
	)
	nr.network.AddWatcher(nr.connectionManager.ConnectionWatcher)

	nr.SetHandleMessageFromClientCheckerFuncs(DefaultHandleMessageFromClientCheckerFuncs...)
	nr.SetHandleBaseBallotCheckerFuncs(DefaultHandleBaseBallotCheckerFuncs...)
	nr.SetHandleINITBallotCheckerFuncs(DefaultHandleINITBallotCheckerFuncs...)
	nr.SetHandleSIGNBallotCheckerFuncs(DefaultHandleSIGNBallotCheckerFuncs...)
	nr.SetHandleACCEPTBallotCheckerFuncs(DefaultHandleACCEPTBallotCheckerFuncs...)

	return
}

func (nr *NodeRunner) Ready() {
	nr.network.SetContext(nr.ctx)
	nr.network.AddHandler(nr.ctx, AddAPIHandlers(nr.storage))
	nr.network.Ready()
}

func (nr *NodeRunner) Start() (err error) {
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
	for t := range ticker.C {
		if !nr.network.IsReady() {
			nr.log.Debug("current network is not ready: %v", t)
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
			if _, ok := err.(sebakcommon.CheckerErrorStop); ok {
				continue
			}
			nr.log.Error("failed to handle sebaknetwork.Message", "message", message.Head(50), "error", err)
		}
	}
}

func (nr *NodeRunner) handleMessageFromClient(message sebaknetwork.Message) (err error) {
	nr.log.Debug("got message`", "message", message.Head(50))

	checker := &NodeRunnerHandleMessageChecker{
		DefaultChecker: sebakcommon.DefaultChecker{nr.handleMessageFromClientCheckerFuncs},
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

	baseChecker := &NodeRunnerHandleBallotChecker{
		DefaultChecker: sebakcommon.DefaultChecker{nr.handleBaseBallotCheckerFuncs},
		NodeRunner:     nr,
		LocalNode:      nr.localNode,
		NetworkID:      nr.networkID,
		Message:        message,
		Log:            nr.Log(),
		VotingHole:     VotingNOTYET,
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

	checker := &NodeRunnerHandleBallotChecker{
		DefaultChecker: sebakcommon.DefaultChecker{checkerFuncs},
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
		if _, ok := err.(sebakcommon.CheckerErrorStop); !ok {
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

	nr.log.Debug("caught up with network and connected to all validators")

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
	candidates := sort.StringSlice(nr.connectionManager.AllValidators())
	candidates.Sort()

	var hashedNumber int
	for _, i := range sebakcommon.MakeHash([]byte(fmt.Sprintf("%d+%d", blockHeight, roundNumber))) {
		hashedNumber += int(i)
	}
	return candidates[hashedNumber%len(candidates)]
}

func (nr *NodeRunner) StartNewRound(roundNumber uint64) {
	if nr.timerExpireRound != nil {
		nr.timerExpireRound.Stop()
		nr.timerExpireRound = nil
	}

	go func() {
		// wait for new ballot from new proposer
		nr.timerExpireRound = time.NewTimer(TimeoutExpireRound)
		go func() {
			for {
				select {
				case <-nr.timerExpireRound.C:
					go nr.StartNewRound(roundNumber + 1)
					return
				}
			}
		}()
	}()

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

	transactionsChecker := &NodeRunnerHandleTransactionChecker{
		DefaultChecker: sebakcommon.DefaultChecker{handleBallotTransactionCheckerFuncs},
		NodeRunner:     nr,
		LocalNode:      nr.localNode,
		NetworkID:      nr.networkID,
		Transactions:   availableTransactions,
		CheckAll:       true,
		VotingHole:     VotingNOTYET,
	}

	{
		err := sebakcommon.RunChecker(transactionsChecker, sebakcommon.DefaultDeferFunc)
		if err != nil {
			if _, ok := err.(sebakcommon.CheckerErrorStop); !ok {
			}
		}
	}

	ballot := NewBallot(nr.localNode, round, transactionsChecker.ValidTransactions)
	ballot.SetVote(sebakcommon.BallotStateINIT, VotingYES)
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

func (nr *NodeRunner) CloseConsensus(round Round, confirmed bool) {
	nr.consensus.SetLatestRound(round)

	if confirmed {
		go nr.StartNewRound(0)
	} else {
		go nr.StartNewRound(round.Number + 1)
	}
}
