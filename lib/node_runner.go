package sebak

import (
	"context"
	"time"

	logging "github.com/inconshreveable/log15"
	"github.com/spikeekips/sebak/lib/network"
	"github.com/spikeekips/sebak/lib/util"
)

type NodeRunner struct {
	currentNode       util.Node
	policy            VotingThresholdPolicy
	transportServer   network.TransportServer
	consensusProtocol Consensus
	connectionManager *network.ConnectionManager

	handleBallotCheckerFuncs            []util.CheckerFunc
	handleMessageFromClientCheckerFuncs []util.CheckerFunc

	handleBallotCheckerFuncsContext            context.Context
	handleMessageFromClientCheckerFuncsContext context.Context

	ctx context.Context
	log logging.Logger
}

func NewNodeRunner(
	currentNode util.Node,
	policy VotingThresholdPolicy,
	transportServer network.TransportServer,
	consensusProtocol Consensus,
) *NodeRunner {
	nr := &NodeRunner{
		currentNode:       currentNode,
		policy:            policy,
		transportServer:   transportServer,
		consensusProtocol: consensusProtocol,
		log:               log.New(logging.Ctx{"node": currentNode.Alias()}),

		handleMessageFromClientCheckerFuncs: DefaultHandleMessageFromClientCheckerFuncs,
		handleBallotCheckerFuncs:            DefaulthandleBallotCheckerFuncs,
	}
	nr.ctx = context.WithValue(context.Background(), "currentNode", currentNode)

	nr.connectionManager = network.NewConnectionManager(
		nr.currentNode,
		nr.transportServer,
		nr.currentNode.GetValidators(),
	)
	nr.transportServer.AddWatcher(nr.connectionManager.ConnectionWatcher)

	return nr
}

func (nr *NodeRunner) Ready() {
	nr.transportServer.SetContext(nr.ctx)
	nr.transportServer.Ready()
}

func (nr *NodeRunner) Start() (err error) {
	nr.Ready()

	go nr.handleMessage()
	go nr.ConnectValidators()

	if err = nr.transportServer.Start(); err != nil {
		return
	}

	return
}

func (nr *NodeRunner) Stop() {
	nr.transportServer.Stop()
}

func (nr *NodeRunner) Node() util.Node {
	return nr.currentNode
}

func (nr *NodeRunner) TransportServer() network.TransportServer {
	return nr.transportServer
}

func (nr *NodeRunner) Consensus() Consensus {
	return nr.consensusProtocol
}

func (nr *NodeRunner) ConnectionManager() *network.ConnectionManager {
	return nr.connectionManager
}

func (nr *NodeRunner) ConnectValidators() {
	ticker := time.NewTicker(time.Millisecond * 5)
	for t := range ticker.C {
		if !nr.transportServer.IsReady() {
			nr.log.Debug("current server is not ready: %v", t)
			continue
		}

		ticker.Stop()
		break
	}
	nr.log.Debug("current server is ready")
	nr.log.Debug("trying to connect to the validators", "validators", nr.currentNode.GetValidators())

	nr.log.Debug("initializing connectionManager for validators")
	nr.connectionManager.Start()
}

var DefaultHandleMessageFromClientCheckerFuncs = []util.CheckerFunc{
	checkNodeRunnerHandleMessageTransactionUnmarshal,
	checkNodeRunnerHandleMessageISAACReceiveMessage,
	checkNodeRunnerHandleMessageSignBallot,
	checkNodeRunnerHandleMessageBroadcast,
}

var DefaulthandleBallotCheckerFuncs = []util.CheckerFunc{
	checkNodeRunnerHandleBallotIsWellformed,
	checkNodeRunnerHandleBallotCheckIsNew,
	checkNodeRunnerHandleBallotReceiveBallot,
	checkNodeRunnerHandleBallotIsClosed,
	checkNodeRunnerHandleBallotBroadcast,
	checkNodeRunnerHandleBallotStore,
}

func (nr *NodeRunner) SetHandleMessageFromClientCheckerFuncs(ctx context.Context, f ...util.CheckerFunc) {
	nr.handleMessageFromClientCheckerFuncs = f

	if ctx == nil {
		ctx = context.Background()
	}
	ctx = context.WithValue(ctx, "currentNode", nr.currentNode)
	nr.handleMessageFromClientCheckerFuncsContext = ctx
}

func (nr *NodeRunner) SetHandleBallotCheckerFuncs(ctx context.Context, f ...util.CheckerFunc) {
	nr.handleBallotCheckerFuncs = f

	if ctx == nil {
		ctx = context.Background()
	}
	ctx = context.WithValue(ctx, "currentNode", nr.currentNode)
	nr.handleBallotCheckerFuncsContext = ctx
}

func (nr *NodeRunner) handleMessage() {
	var err error
	for message := range nr.transportServer.ReceiveMessage() {
		switch message.Type {
		case network.ConnectMessage:
			nr.log.Debug("got `ConnectMessage`", "message", message)
			if _, err := util.NewValidatorFromString(message.Data); err != nil {
				nr.log.Error("invalid validator data was received", "data", message.Data)
				continue
			}
		case network.MessageFromClient:
			nr.log.Debug("got `MessageFromClient`", "message", message)

			/*
				- TODO `Message` must be saved in `BlockTransactionHistory`
				- TODO check already `IsWellFormed()`
				- TODO check already in BlockTransaction
				- TODO check already in BlockTransactionHistory
			*/

			if _, err = util.Checker(nr.handleMessageFromClientCheckerFuncsContext, nr.handleMessageFromClientCheckerFuncs...)(nr, message); err != nil {
				if _, ok := err.(util.CheckerErrorStop); ok {
					continue
				}
				nr.log.Error("failed to handle message from client", "error", err)
				continue
			}
		case network.BallotMessage:
			nr.log.Debug("got `Ballot`", "message", message)
			/*
				- TODO check already `IsWellFormed()`
				- TODO check already in BlockTransaction
				- TODO check already in BlockTransactionHistory
			*/

			if _, err = util.Checker(nr.handleBallotCheckerFuncsContext, nr.handleBallotCheckerFuncs...)(nr, message); err != nil {
				if _, ok := err.(util.CheckerErrorStop); ok {
					continue
				}
				nr.log.Error("failed to handle ballot", "error", err)
				continue
			}
		}
	}
}
