package consensus

import (
	"boscoin.io/sebak/lib"
	"boscoin.io/sebak/pkg/wire/message"
)

type IsaacReceiver struct {
	isaac *sebak.ISAAC
}

func NewIsaacReceiver() *IsaacReceiver {
	// networkID []byte, node *sebaknode.LocalNode, votingThresholdPolicy sebakcommon.VotingThresholdPolicy
	//isaac := sebak.NewISAAC()
	return &IsaacReceiver{
		isaac: nil,
	}
}

func (o *IsaacReceiver) Start() {
}

func (o *IsaacReceiver) Stop() {
}

func (o *IsaacReceiver) OnConnect(id message.PeerId) {
}

func (o *IsaacReceiver) Receive(id message.PeerId, msg interface{}) {
}
