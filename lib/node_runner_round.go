package sebak

import (
	"context"
	"errors"
	"time"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/network"
	"boscoin.io/sebak/lib/node"
	"boscoin.io/sebak/lib/storage"
	logging "github.com/inconshreveable/log15"
)

var (
	TimeoutExpireRound           time.Duration = time.Second * 10
	TimeoutProposeNewRoundBallot time.Duration = time.Second * 2
	MaxTransactionsInRoundBallot int           = 1000
)

type NodeRunnerRound struct {
	networkID         []byte
	localNode         *sebaknode.LocalNode
	policy            sebakcommon.VotingThresholdPolicy
	network           sebaknetwork.Network
	consensus         *ISAACRound
	connectionManager *sebaknetwork.ConnectionManager
	storage           *sebakstorage.LevelDBBackend

	handleMessageFromClientCheckerFuncs []sebakcommon.CheckerFunc
	handleBallotCheckerFuncs            []sebakcommon.CheckerFunc
	handleRoundBallotCheckerFuncs       []sebakcommon.CheckerFunc

	handleMessageFromClientCheckerDeferFunc sebakcommon.CheckerDeferFunc
	handleBallotCheckerDeferFunc            sebakcommon.CheckerDeferFunc
	handleBallotFinishedFunc                sebakcommon.CheckerDeferFunc
	handleRoundBallotCheckerDeferFunc       sebakcommon.CheckerDeferFunc

	ctx context.Context
	log logging.Logger

	timerExpireRound *time.Timer
}

func NewNodeRunnerRound(
	networkID string,
	localNode *sebaknode.LocalNode,
	policy sebakcommon.VotingThresholdPolicy,
	network sebaknetwork.Network,
	consensus *ISAACRound,
	storage *sebakstorage.LevelDBBackend,
) (nr *NodeRunnerRound, err error) {
	nr = &NodeRunnerRound{
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

	nr.SetHandleMessageFromClientCheckerFuncs(nil, DefaultRoundHandleMessageFromClientCheckerFuncs...)
	nr.SetHandleBallotFuncs(nil, nil, DefaultRoundHandleBallotCheckerFuncs...)
	nr.SetHandleRoundBallotCheckerFuncs(nil, nil, DefaultRoundHandleRoundBallotCheckerFuncs...)

	return
}

func (nr *NodeRunnerRound) Ready() {
	nr.network.SetContext(nr.ctx)
	nr.network.AddHandler(nr.ctx, AddAPIHandlers(nr.storage))
	nr.network.Ready()
}

func (nr *NodeRunnerRound) Start() (err error) {
	nr.Ready()

	go nr.handleMessage()
	go nr.ConnectValidators()
	go nr.StartRound()

	if err = nr.network.Start(); err != nil {
		return
	}

	return
}

func (nr *NodeRunnerRound) Stop() {
	nr.network.Stop()
}

func (nr *NodeRunnerRound) Node() sebaknode.Node {
	return nr.localNode
}

func (nr *NodeRunnerRound) NetworkID() []byte {
	return nr.networkID
}

func (nr *NodeRunnerRound) Network() sebaknetwork.Network {
	return nr.network
}

func (nr *NodeRunnerRound) Consensus() *ISAACRound {
	return nr.consensus
}

func (nr *NodeRunnerRound) ConnectionManager() *sebaknetwork.ConnectionManager {
	return nr.connectionManager
}

func (nr *NodeRunnerRound) Storage() *sebakstorage.LevelDBBackend {
	return nr.storage
}

func (nr *NodeRunnerRound) Policy() sebakcommon.VotingThresholdPolicy {
	return nr.policy
}

func (nr *NodeRunnerRound) Log() logging.Logger {
	return nr.log
}

func (nr *NodeRunnerRound) ConnectValidators() {
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

var DefaultRoundHandleMessageFromClientCheckerFuncs = []sebakcommon.CheckerFunc{
	CheckNodeRunnerRoundHandleMessageTransactionUnmarshal,
	CheckNodeRunnerRoundHandleMessageHistory,
	CheckNodeRunnerRoundHandleMessageISAACReceiveMessage,
	CheckNodeRunnerRoundHandleMessageSignBallot,
	CheckNodeRunnerRoundHandleMessageBroadcast,
}

var DefaultRoundHandleBallotCheckerFuncs = []sebakcommon.CheckerFunc{
	CheckNodeRunnerRoundHandleBallotIsWellformed,
	CheckNodeRunnerRoundHandleBallotNotFromKnownValidators,
	CheckNodeRunnerRoundHandleBallotCheckIsNew,
	CheckNodeRunnerRoundHandleBallotReceiveBallot,
	CheckNodeRunnerRoundHandleBallotReachedToSIGN,
	CheckNodeRunnerRoundHandleBallotHistory,
	CheckNodeRunnerRoundHandleBallotIsBroadcastable,
	CheckNodeRunnerRoundHandleBallotBroadcast,
}

var DefaultRoundHandleRoundBallotCheckerFuncs = []sebakcommon.CheckerFunc{
	CheckNodeRunnerRoundHandleRoundBallotUnmarshal,
	CheckNodeRunnerRoundHandleRoundBallotAlreadyFinished,
	CheckNodeRunnerRoundHandleRoundBallotAlreadyVoted,
	CheckNodeRunnerRoundHandleRoundBallotAddRunningRounds,
	CheckNodeRunnerRoundHandleRoundBallotIsSameProposer,
	CheckNodeRunnerRoundHandleRoundBallotValidateTransactions,
	CheckNodeRunnerRoundHandleRoundBallotBroadcast,
	CheckNodeRunnerRoundHandleRoundBallotStore,
}

func (nr *NodeRunnerRound) SetHandleMessageFromClientCheckerFuncs(
	deferFunc sebakcommon.CheckerDeferFunc,
	f ...sebakcommon.CheckerFunc,
) {
	if len(f) > 0 {
		nr.handleMessageFromClientCheckerFuncs = f
	}

	if deferFunc == nil {
		deferFunc = sebakcommon.DefaultDeferFunc
	}

	nr.handleMessageFromClientCheckerDeferFunc = deferFunc
}

func (nr *NodeRunnerRound) SetHandleBallotFuncs(
	deferFunc sebakcommon.CheckerDeferFunc,
	finishedFunc sebakcommon.CheckerDeferFunc,
	f ...sebakcommon.CheckerFunc,
) {
	if len(f) > 0 {
		nr.handleBallotCheckerFuncs = f
	}

	if deferFunc == nil {
		deferFunc = sebakcommon.DefaultDeferFunc
	}

	if finishedFunc == nil {
		finishedFunc = sebakcommon.DefaultDeferFunc
	}

	nr.handleBallotCheckerDeferFunc = deferFunc
	nr.handleBallotFinishedFunc = finishedFunc
}

func (nr *NodeRunnerRound) SetHandleRoundBallotCheckerFuncs(
	deferFunc sebakcommon.CheckerDeferFunc,
	finishedFunc sebakcommon.CheckerDeferFunc,
	f ...sebakcommon.CheckerFunc,
) {
	if len(f) > 0 {
		nr.handleRoundBallotCheckerFuncs = f
	}

	if deferFunc == nil {
		deferFunc = sebakcommon.DefaultDeferFunc
	}

	if finishedFunc == nil {
		finishedFunc = sebakcommon.DefaultDeferFunc
	}

	nr.handleRoundBallotCheckerDeferFunc = deferFunc
}

func (nr *NodeRunnerRound) handleMessage() {
	var err error
	for message := range nr.network.ReceiveMessage() {
		if message.IsEmpty() {
			nr.log.Error("got empty message`")
			continue
		}
		switch message.Type {
		case sebaknetwork.ConnectMessage:
			nr.log.Debug("got connect", "message", message.Head(50))
			if _, err := sebaknode.NewValidatorFromString(message.Data); err != nil {
				err = errors.New("invalid validator data was received")
			}
		case sebaknetwork.MessageFromClient:
			err = nr.handleMessageFromClient(message)
		case sebaknetwork.BallotMessage:
			err = nr.handleBallotMessage(message)
		case sebaknetwork.RoundBallotMessage:
			err = nr.handleRoundBallotMessage(message)
		default:
			err = errors.New("got unknown message")
		}

		if err != nil {
			if _, ok := err.(sebakcommon.CheckerErrorStop); ok {
				nr.log.Debug("consensus finished", "message", message.Head(50), "error", err)
				continue
			}
			nr.log.Debug("failed to handle sebaknetwork.Message", "message", message.Head(50), "error", err)
		}
	}
}

func (nr *NodeRunnerRound) handleMessageFromClient(message sebaknetwork.Message) (err error) {
	nr.log.Debug("got message from client`", "message", message.Head(50))

	checker := &NodeRunnerRoundHandleMessageChecker{
		DefaultChecker: sebakcommon.DefaultChecker{nr.handleMessageFromClientCheckerFuncs},
		NodeRunner:     nr,
		LocalNode:      nr.localNode,
		NetworkID:      nr.networkID,
		Message:        message,
	}

	if err = sebakcommon.RunChecker(checker, nr.handleMessageFromClientCheckerDeferFunc); err != nil {
		if _, ok := err.(sebakcommon.CheckerErrorStop); ok {
			return
		}
		nr.log.Error("failed to handle message from client", "error", err)
		return
	}

	return
}

func (nr *NodeRunnerRound) handleBallotMessage(message sebaknetwork.Message) (err error) {
	nr.log.Debug("got ballot", "message", message.Head(50))

	checker := &NodeRunnerRoundHandleBallotChecker{
		GenesisBlockCheckpoint: sebakcommon.MakeGenesisCheckpoint(nr.networkID),
		DefaultChecker:         sebakcommon.DefaultChecker{nr.handleBallotCheckerFuncs},
		NodeRunner:             nr,
		LocalNode:              nr.localNode,
		NetworkID:              nr.networkID,
		Message:                message,
		VotingHole:             VotingNOTYET,
	}

	if err = sebakcommon.RunChecker(checker, nr.handleBallotCheckerDeferFunc); err != nil {
		if _, ok := err.(sebakcommon.CheckerErrorStop); ok {
			nr.log.Debug("stop handling ballot", "reason", err)
		} else {
			nr.log.Debug("failed to handle ballot", "error", err)
		}
	}

	if err == nil {
		return
	}

	defer func(err error) {
		nr.handleBallotFinishedFunc(0, checker, err)
		return
	}(err)

	if checker.VotingStateStaging.IsEmpty() {
		return
	}
	if checker.VotingStateStaging.State < sebakcommon.BallotStateSIGN {
		return
	}

	if err = nr.Consensus().CloseBallotConsensus(checker.Ballot); err != nil {
		nr.Log().Error("new failed to close consensus", "error", err)
		return
	}

	nr.Log().Debug("ballot consensus closed")

	return
}

func (nr *NodeRunnerRound) handleRoundBallotMessage(message sebaknetwork.Message) (err error) {
	nr.log.Debug("got round-ballot", "message", message.Head(50))

	checker := &NodeRunnerRoundHandleRoundBallotChecker{
		GenesisBlockCheckpoint: sebakcommon.MakeGenesisCheckpoint(nr.networkID),
		DefaultChecker:         sebakcommon.DefaultChecker{nr.handleRoundBallotCheckerFuncs},
		NodeRunner:             nr,
		LocalNode:              nr.localNode,
		NetworkID:              nr.networkID,
		Message:                message,
		VotingHole:             VotingNOTYET,
	}
	err = sebakcommon.RunChecker(checker, nr.handleRoundBallotCheckerDeferFunc)
	if err != nil {
		if _, ok := err.(sebakcommon.CheckerErrorStop); ok {
			nr.log.Debug("stop handling round-ballot", "reason", err)
		} else {
			nr.log.Debug("failed to handle round-ballot", "error", err)
		}
	}

	return
}

func (nr *NodeRunnerRound) StartRound() {
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

	nr.log.Debug("all validators are checked for connectivity")

	go nr.startRound()
}

func (nr *NodeRunnerRound) startRound() {
	// check whether current running rounds exist
	if len(nr.Consensus().RunningRounds) > 0 {
		return
	}

	nr.StartNewRound(0)
}

func (nr *NodeRunnerRound) StartNewRound(roundNumber uint64) {
	if nr.timerExpireRound != nil {
		nr.timerExpireRound.Stop()
		nr.timerExpireRound = nil
	}

	candidates := nr.connectionManager.RoundCandidates()
	proposer := nr.Consensus().CalculateProposer(
		candidates,
		nr.Consensus().LatestConfirmedBlock.Height,
		roundNumber,
	)
	log.Debug("calculated proposer", "proposer", proposer, "candidates", candidates)

	if proposer != nr.Node().Address() {
		// wait for new RoundBallot from new proposer
		nr.timerExpireRound = time.NewTimer(TimeoutExpireRound)
		go func() {
			for {
				select {
				case <-nr.timerExpireRound.C:
					if !nr.Consensus().IsRunningRound(roundNumber) {
						go nr.StartNewRound(roundNumber + 1)
					}
					goto end
				}
			}

		end:
			//
		}()

		return
	}

	nr.readyToProposeNewRoundBallot(roundNumber)

	return
}

func (nr *NodeRunnerRound) readyToProposeNewRoundBallot(roundNumber uint64) {
	// if incoming transaactions are over `MaxTransactionsInRoundBallot`, just
	// start.
	if len(nr.Consensus().TransactionPoolHashes) > MaxTransactionsInRoundBallot {
		nr.proposeNewRoundBallot(roundNumber)
		return
	}

	timer := time.NewTimer(TimeoutProposeNewRoundBallot)
	go func() {
		<-timer.C

		nr.proposeNewRoundBallot(roundNumber)
	}()

	return
}

func (nr *NodeRunnerRound) proposeNewRoundBallot(roundNumber uint64) {
	// start new round
	round := Round{
		Number:      roundNumber,
		BlockHeight: nr.Consensus().LatestConfirmedBlock.Height,
		BlockHash:   nr.Consensus().LatestConfirmedBlock.Hash,
		TotalTxs:    nr.Consensus().LatestConfirmedBlock.TotalTxs,
	}

	// collect incoming transactions from `TransactionPool`
	nr.log.Debug("new round proposed", "round", round)

	roundBallot := NewRoundBallot(
		nr.localNode,
		round,
		nr.Consensus().AvailableTransactions(),
	)

	// TODO validate transactions
	roundBallot.SetValidTransactions(nr.Consensus().TransactionPoolHashes)
	roundBallot.SetVote(VotingYES)
	roundBallot.Sign(nr.localNode.Keypair(), nr.networkID)

	nr.log.Debug("new RoundBallot created", "roundBallot", roundBallot)

	nr.ConnectionManager().Broadcast(roundBallot)

	runningRound := NewRunningRound(nr.localNode.Address(), *roundBallot)
	rr := nr.Consensus().RunningRounds
	rr[round.Hash()] = runningRound

	nr.Log().Debug("round-ballot broadcasted and voted", "runningRound", runningRound)

	return
}

func (nr *NodeRunnerRound) CloseConsensus(round Round, confirmed bool) {
	nr.Consensus().SetLatestRound(round)

	if confirmed {
		go nr.StartNewRound(0)
	} else {
		go nr.StartNewRound(round.Number + 1)
	}
}
