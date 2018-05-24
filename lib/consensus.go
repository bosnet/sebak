package sebak

import (
	"github.com/spikeekips/sebak/lib/util"
)

type Consensus interface {
	GetNode() util.Node
	HasMessage(util.Message) bool
	ReceiveMessage(util.Message) (Ballot, error)
	ReceiveBallot(Ballot) (VotingStateStaging, error)
}
