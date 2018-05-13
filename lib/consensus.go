package sebak

import (
	"github.com/spikeekips/sebak/lib/util"
)

type Consensus interface {
	GetNode() util.Node
	ReceiveMessage(util.Message) (Ballot, error)
	ReceiveBallot(Ballot) (VotingStateStaging, error)
}
