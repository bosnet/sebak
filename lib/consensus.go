package sebak

import (
	"github.com/spikeekips/sebak/lib/common"
)

type Consensus interface {
	GetNode() sebakcommon.Node
	HasMessage(sebakcommon.Message) bool
	HasMessageByHash(string) bool
	ReceiveMessage(sebakcommon.Message) (Ballot, error)
	ReceiveBallot(Ballot) (VotingStateStaging, error)

	AddBallot(Ballot) error
	CloseConsensus(Ballot) error
}
