package sebak

import (
	"context"
	"time"

	logging "github.com/inconshreveable/log15"
	"github.com/spikeekips/sebak/lib/common"
	"github.com/spikeekips/sebak/lib/network"
	"github.com/spikeekips/sebak/lib/storage"
)

type NodeRunner struct {
	currentNode       sebakcommon.Node
	policy            sebakcommon.VotingThresholdPolicy
	network           sebaknetwork.Network
	consensus         Consensus
	connectionManager *sebaknetwork.ConnectionManager
	storage           *sebakstorage.LevelDBBackend

	handleBallotCheckerFuncs            []sebakcommon.CheckerFunc
	handleMessageFromClientCheckerFuncs []sebakcommon.CheckerFunc

	handleBallotCheckerFuncsContext            context.Context
	handleMessageFromClientCheckerFuncsContext context.Context

	ctx context.Context
	log logging.Logger
}

func NewNodeRunner(
	currentNode sebakcommon.Node,
	policy sebakcommon.VotingThresholdPolicy,
	network sebaknetwork.Network,
	consensus Consensus,
	storage *sebakstorage.LevelDBBackend,
) *NodeRunner {
	nr := &NodeRunner{
		currentNode: currentNode,
		policy:      policy,
		network:     network,
		consensus:   consensus,
		storage:     storage,
		log:         log.New(logging.Ctx{"node": currentNode.Alias()}),
	}
	nr.ctx = context.WithValue(context.Background(), "currentNode", currentNode)

	nr.connectionManager = sebaknetwork.NewConnectionManager(
		nr.currentNode,
		nr.network,
		nr.policy,
		nr.currentNode.GetValidators(),
	)
	nr.network.AddWatcher(nr.connectionManager.ConnectionWatcher)

	nr.SetHandleMessageFromClientCheckerFuncs(nil, DefaultHandleMessageFromClientCheckerFuncs...)
	nr.SetHandleBallotCheckerFuncs(nil, DefaultHandleBallotCheckerFuncs...)
	return nr
}

func (nr *NodeRunner) Ready() {
	nr.network.SetContext(nr.ctx)
	nr.network.Ready()
}

func (nr *NodeRunner) Start() (err error) {
	nr.Ready()

	go nr.handleMessage()
	go nr.ConnectValidators()

	if err = nr.network.Start(); err != nil {
		return
	}

	return
}

func (nr *NodeRunner) Stop() {
	nr.network.Stop()
}

func (nr *NodeRunner) Node() sebakcommon.Node {
	return nr.currentNode
}

func (nr *NodeRunner) Network() sebaknetwork.Network {
	return nr.network
}

func (nr *NodeRunner) Consensus() Consensus {
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
	nr.log.Debug("trying to connect to the validators", "validators", nr.currentNode.GetValidators())

	nr.log.Debug("initializing connectionManager for validators")
	nr.connectionManager.Start()
}

var DefaultHandleMessageFromClientCheckerFuncs = []sebakcommon.CheckerFunc{
	CheckNodeRunnerHandleMessageTransactionUnmarshal,
	CheckNodeRunnerHandleMessageHistory,
	CheckNodeRunnerHandleMessageISAACReceiveMessage,
	CheckNodeRunnerHandleMessageSignBallot,
	CheckNodeRunnerHandleMessageBroadcast,
}

var DefaultHandleBallotCheckerFuncs = []sebakcommon.CheckerFunc{
	CheckNodeRunnerHandleBallotIsWellformed,
	CheckNodeRunnerHandleBallotCheckIsNew,
	CheckNodeRunnerHandleBallotReceiveBallot,
	CheckNodeRunnerHandleBallotHistory,
	CheckNodeRunnerHandleBallotStore,
	CheckNodeRunnerHandleBallotBroadcast,
}

func (nr *NodeRunner) SetHandleMessageFromClientCheckerFuncs(ctx context.Context, f ...sebakcommon.CheckerFunc) {
	nr.handleMessageFromClientCheckerFuncs = f

	if ctx == nil {
		ctx = context.Background()
	}
	ctx = context.WithValue(ctx, "currentNode", nr.currentNode)
	nr.handleMessageFromClientCheckerFuncsContext = ctx
}

func (nr *NodeRunner) SetHandleBallotCheckerFuncs(ctx context.Context, f ...sebakcommon.CheckerFunc) {
	if len(f) > 0 {
		nr.handleBallotCheckerFuncs = f
	}

	if ctx == nil {
		ctx = context.Background()
	}
	ctx = context.WithValue(ctx, "currentNode", nr.currentNode)
	nr.handleBallotCheckerFuncsContext = ctx
}

func (nr *NodeRunner) handleMessage() {
	var err error
	for message := range nr.network.ReceiveMessage() {
		switch message.Type {
		case sebaknetwork.ConnectMessage:
			nr.log.Debug("got connect", "message", message.String()[:50])
			if _, err := sebakcommon.NewValidatorFromString(message.Data); err != nil {
				nr.log.Error("invalid validator data was received", "data", message.Data)
				continue
			}
		case sebaknetwork.MessageFromClient:
			nr.log.Debug("got message from client`", "message", message.String()[:50])

			/*
				- TODO check already `IsWellFormed()`
				- TODO check already in BlockTransaction
				- TODO check already in BlockTransactionHistory
			*/

			if _, err = sebakcommon.Checker(nr.handleMessageFromClientCheckerFuncsContext, nr.handleMessageFromClientCheckerFuncs...)(nr, message); err != nil {
				if _, ok := err.(sebakcommon.CheckerErrorStop); ok {
					continue
				}
				nr.log.Error("failed to handle message from client", "error", err)
				continue
			}
		case sebaknetwork.BallotMessage:
			nr.log.Debug("got ballot", "message", message.String()[:50])
			/*
				- TODO check already `IsWellFormed()`
				- TODO check already in BlockTransaction
				- TODO check already in BlockTransactionHistory
			*/

			if _, err = sebakcommon.Checker(nr.handleBallotCheckerFuncsContext, nr.handleBallotCheckerFuncs...)(nr, message); err != nil {
				if _, ok := err.(sebakcommon.CheckerErrorStop); ok {
					continue
				}
				nr.log.Error("failed to handle ballot", "error", err)
				continue
			}
		}
	}
}
