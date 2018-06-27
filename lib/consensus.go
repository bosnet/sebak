package sebak

import (
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/node"
)

type Consensus interface {
	GetNode() *sebaknode.Node
	HasMessage(sebakcommon.Message) bool
	HasMessageByHash(string) bool
	ReceiveMessage(sebakcommon.Message) (Ballot, error)
	ReceiveBallot(Ballot) (VotingStateStaging, error)

	AddBallot(Ballot) error
	CloseConsensus(Ballot) error
}
