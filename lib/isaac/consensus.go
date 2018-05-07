package consensus

import (
	"github.com/spikeekips/sebak/lib"
	"github.com/spikeekips/sebak/lib/util"
)

type Consensus interface {
	GetNode() sebak.Node
	ReceiveMessage(util.Message) (Ballot, error)
	ReceiveBallot(Ballot) (*VotingResult, error)
}
