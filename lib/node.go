package sebak

import (
	"context"
	"fmt"

	"github.com/spikeekips/sebak/lib/network"
	"github.com/spikeekips/sebak/lib/util"
)

type NodeRunner struct {
	currentNode       util.Node
	policy            VotingThresholdPolicy
	transportServer   network.TransportServer
	consensusProtocol Consensus

	ctx context.Context
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
	}
	ctx := context.WithValue(context.Background(), "currentNode", currentNode)
	nr.ctx = ctx

	return nr

}

func (nr *NodeRunner) Ready() {
	nr.transportServer.SetContext(nr.ctx)
	nr.transportServer.Ready()
}

func (nr *NodeRunner) Start() (err error) {
	go nr.handleMessage()

	if err = nr.transportServer.Start(); err != nil {
		return
	}

	return
}

func (nr *NodeRunner) handleMessage() {
	var err error
	for message := range nr.transportServer.ReceiveMessage() {
		log.Debug("got message", "message", message)

		switch message.Type {
		case "message":
			var tx Transaction
			if tx, err = NewTransactionFromJSON(message.Data); err != nil {
				log.Error("found invalid transaction message", "error", err)

				// TODO if failed, save in `BlockTransactionHistory`????
				continue
			}
			if err = tx.IsWellFormed(); err != nil {
				log.Error("found invalid transaction message", "error", err)
				// TODO if failed, save in `BlockTransactionHistory`
				continue
			}

			/*
				- TODO `Message` must be saved in `BlockTransactionHistory`
				- TODO check already `IsWellFormed()`
				- TODO check already in BlockTransaction
				- TODO check already in BlockTransactionHistory
			*/

			var ballot Ballot
			if ballot, err = nr.consensusProtocol.ReceiveMessage(tx); err != nil {
				log.Error("failed to receive new message", "error", err)
				continue
			}

			// TODO initially shutup and broadcast
			fmt.Println(ballot)
		case "ballot":
			/*
				- TODO check already `IsWellFormed()`
				- TODO check already in BlockTransaction
				- TODO check already in BlockTransactionHistory
			*/

			var ballot Ballot
			if ballot, err = NewBallotFromJSON(message.Data); err != nil {
				log.Error("found invalid ballot message", "error", err)
				continue
			}
			var vt VotingStateStaging
			if vt, err = nr.consensusProtocol.ReceiveBallot(ballot); err != nil {
				log.Error("failed to receive ballot", "error", err)
				continue
			}

			if vt.IsEmpty() {
				continue
			}

			if vt.IsClosed() {
				if !vt.IsStorable() {
					continue
				}
				// store in BlockTransaction
			}

			if !vt.IsChanged() {
				continue
			}

			// TODO state is changed, so broadcast

			fmt.Println(vt)
		}
	}
}
