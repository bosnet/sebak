package sebak

import (
	"errors"
	"sort"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/error"
	"boscoin.io/sebak/lib/node"
)

type RoundVote map[ /* Node.Address() */ string]VotingHole

func (rv RoundVote) CanGetVotingResult(policy sebakcommon.VotingThresholdPolicy) (VotingHole, bool) {
	threshold := policy.Threshold(sebakcommon.BallotStateACCEPT)
	if threshold < 1 {
		return VotingNOTYET, false
	}
	if len(rv) < int(threshold) {
		return VotingNOTYET, false
	}

	var yes int
	var no int
	for _, vh := range rv {
		if vh == VotingYES {
			yes++
		} else if vh == VotingNO {
			no++
		}
	}

	log.Debug(
		"check threshold",
		"threshold", threshold,
		"yes", yes,
		"no", no,
		"policy", policy,
	)

	if yes >= threshold {
		return VotingYES, true
	} else if no >= threshold {
		return VotingNO, true
	}

	// check draw!
	total := policy.Validators()
	voted := yes + no
	if total-voted < threshold-yes && total-voted < threshold-no { // draw
		return VotingNO, true
	}

	return VotingNOTYET, false
}

type RunningRound struct {
	sebakcommon.SafeLock

	Round        Round
	Proposer     string                              // LocalNode's `Proposer`
	Transactions map[ /* Proposer */ string][]string /* Transaction.Hash */
	Voted        map[ /* Proposer */ string]RoundVote
}

func NewRunningRound(proposer string, rb RoundBallot) *RunningRound {
	transactions := map[string][]string{
		rb.Proposer(): rb.Transactions(),
	}

	voted := map[string]RoundVote{
		rb.Proposer(): RoundVote{
			rb.Source(): rb.Vote(),
		},
	}

	if !rb.IsFromProposer() {
		voted[rb.Proposer()][rb.Proposer()] = VotingYES
	}

	return &RunningRound{
		Round:        rb.Round(),
		Transactions: transactions,
		Voted:        voted,
	}
}

func (rr *RunningRound) RoundVote(proposer string) (rv RoundVote, err error) {
	var found bool
	rv, found = rr.Voted[proposer]
	if !found {
		err = sebakerror.ErrorRoundVoteNotFound
		return
	}
	return
}

func (rr *RunningRound) IsVoted(rb RoundBallot) bool {
	roundVote, err := rr.RoundVote(rb.Proposer())
	if err != nil {
		return false
	}

	_, voted := roundVote[rb.Source()]
	return voted
}

func (rr *RunningRound) Vote(rb RoundBallot) (isNew bool) {
	rr.Lock()
	defer rr.Unlock()

	if _, found := rr.Voted[rb.Proposer()]; !found {
		rr.Voted[rb.Proposer()] = RoundVote{}
		isNew = true
	}

	rr.Voted[rb.Proposer()][rb.Source()] = rb.Vote()
	return
}

type ISAACRound struct {
	sebakcommon.SafeLock

	NetworkID             []byte
	Node                  *sebaknode.LocalNode
	VotingThresholdPolicy sebakcommon.VotingThresholdPolicy
	TransactionPool       map[ /* Transaction.GetHash() */ string]Transaction
	TransactionPoolHashes []string // Transaction.GetHash()
	RunningRounds         map[ /* Round.Hash() */ string]*RunningRound
	LatestConfirmedBlock  Block
	LatestRound           Round

	Boxes *BallotBoxes
}

func NewISAACRound(networkID []byte, node *sebaknode.LocalNode, votingThresholdPolicy sebakcommon.VotingThresholdPolicy) (is *ISAACRound, err error) {
	is = &ISAACRound{
		NetworkID: networkID,
		Node:      node,
		VotingThresholdPolicy: votingThresholdPolicy,
		TransactionPool:       map[string]Transaction{},
		RunningRounds:         map[string]*RunningRound{},
		Boxes:                 NewBallotBoxes(),
	}

	return
}

func (is *ISAACRound) CalculateProposer(connected []string, blockHeight uint64, roundNumber uint64) string {
	is.Lock()
	defer is.Unlock()

	addresses := sort.StringSlice(connected)
	addresses.Sort()

	// TODO This is simple version to select proposer node.
	return addresses[(blockHeight+roundNumber)%uint64(len(addresses))]
}

func (is *ISAACRound) ReceiveMessage(m sebakcommon.Message) (ballot Ballot, err error) {
	if is.Boxes.HasMessage(m) {
		err = sebakerror.ErrorNewButKnownMessage
		return
	}

	if ballot, err = NewBallotFromMessage(is.Node.Address(), m); err != nil {
		return
	}

	// self-sign; make new `Ballot` from `Message`
	ballot.SetState(sebakcommon.BallotStateINIT)
	ballot.Vote(VotingYES) // The initial ballot from client will have 'VotingYES'
	ballot.Sign(is.Node.Keypair(), is.NetworkID)

	if err = ballot.IsWellFormed(is.NetworkID); err != nil {
		return
	}

	if _, err = is.Boxes.AddBallot(ballot); err != nil {
		return
	}

	return
}

func (is *ISAACRound) ReceiveBallot(ballot Ballot) (vs VotingStateStaging, err error) {
	switch ballot.State() {
	case sebakcommon.BallotStateINIT:
		vs, err = is.receiveBallotStateINIT(ballot)
	case sebakcommon.BallotStateALLCONFIRM:
		err = sebakerror.ErrorBallotHasInvalidState
	default:
		err = sebakerror.ErrorBallotHasInvalidState
		//vs, err = is.receiveBallotVotingStates(ballot)
	}

	return
}

func (is *ISAACRound) receiveBallotStateINIT(ballot Ballot) (vs VotingStateStaging, err error) {
	var isNew bool

	if isNew, err = is.Boxes.AddBallot(ballot); err != nil {
		return
	}

	if isNew {
		var newBallot Ballot
		newBallot, err = NewBallotFromMessage(is.Node.Keypair().Address(), ballot.Data().Message())
		if err != nil {
			return
		}

		// self-sign
		newBallot.SetState(sebakcommon.BallotStateINIT)
		newBallot.Vote(VotingYES) // The BallotStateINIT ballot will have 'VotingYES'
		newBallot.Sign(is.Node.Keypair(), is.NetworkID)

		if err = newBallot.IsWellFormed(is.NetworkID); err != nil {
			return
		}

		if _, err = is.Boxes.AddBallot(newBallot); err != nil {
			return
		}
	}

	vr, err := is.Boxes.VotingResult(ballot)
	if err != nil {
		return
	}

	if vr.IsClosed() || !vr.CanGetResult(is.VotingThresholdPolicy) {
		return
	}

	votingHole, state, ended := vr.MakeResult(is.VotingThresholdPolicy)
	if ended {
		if vs, err = vr.ChangeState(votingHole, state); err != nil {
			return
		}
	}

	return
}

func (is *ISAACRound) CloseRoundBallotConsensus(proposer string, round Round, vh VotingHole) (err error) {
	is.Lock()
	defer is.Unlock()

	if vh == VotingNOTYET {
		err = errors.New("invalid VotingHole, `VotingNOTYET`")
		return
	}

	roundHash := round.Hash()
	rr, found := is.RunningRounds[roundHash]
	if !found {
		return
	}

	if vh == VotingNO {
		delete(rr.Transactions, proposer)
		delete(rr.Voted, proposer)

		return
	}

	for _, txHash := range rr.Transactions[proposer] {
		var index int
		var found bool
		if index, found = sebakcommon.InStringArray(is.TransactionPoolHashes, txHash); !found {
			continue
		}
		is.TransactionPoolHashes = append(is.TransactionPoolHashes[:index], is.TransactionPoolHashes[index+1:]...)
		delete(is.TransactionPool, txHash)
	}

	delete(is.RunningRounds, roundHash)

	return
}

func (is *ISAACRound) CloseBallotConsensus(ballot Ballot) (err error) {
	log.Debug("consensus of this ballot will be closed", "ballot", ballot.MessageHash())
	if !is.Boxes.HasMessageByHash(ballot.MessageHash()) {
		return sebakerror.ErrorVotingResultNotInBox
	}

	vr, err := is.Boxes.VotingResult(ballot)
	if err != nil {
		return
	}

	var found bool
	var message sebakcommon.Message
	if message, found = is.Boxes.Messages[ballot.MessageHash()]; !found {
		return
	}

	is.Lock()
	defer is.Unlock()

	tx := message.(Transaction)
	is.TransactionPool[tx.GetHash()] = tx
	is.TransactionPoolHashes = append(is.TransactionPoolHashes, tx.GetHash())

	is.Boxes.RemoveVotingResult(vr) // TODO detect error

	return
}

func (is *ISAACRound) SetLatestConsensusedBlock(block Block) {
	is.LatestConfirmedBlock = block
}

func (is *ISAACRound) SetLatestRound(round Round) {
	is.LatestRound = round
}

func (is *ISAACRound) IsAvailableRound(round Round) bool {
	if round.BlockHeight != is.LatestConfirmedBlock.Height {
		return false
	}

	if is.LatestRound.BlockHash == "" {
		return true
	}

	if round.BlockHeight == is.LatestRound.BlockHeight {
		if round.Number <= is.LatestRound.Number {
			return false
		}
	}

	return true
}
