package sebak

import (
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/node"
)

type Consensus interface {
	GetNode() *sebaknode.LocalNode
	HasMessage(sebakcommon.Message) bool
	HasMessageByHash(string) bool
	ReceiveMessage(sebakcommon.Message) error
	//ReceiveBallot(Ballot) (VotingStateStaging, error)

	//AddBallot(Ballot) error
	//CloseConsensus(Ballot) error
}
