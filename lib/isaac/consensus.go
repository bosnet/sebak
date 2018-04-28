package consensus

import "github.com/spikeekips/sebak/lib/util"

type Consensus interface {
	GetNode() Node
	ReceiveMessage(util.Message) (Ballot, error)
}
