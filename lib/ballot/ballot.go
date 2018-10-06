package ballot

import (
	"encoding/json"
	"time"

	"github.com/btcsuite/btcutil/base58"
	"github.com/stellar/go/keypair"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/consensus/round"
	"boscoin.io/sebak/lib/error"
)

type Ballot struct {
	H BallotHeader
	B BallotBody
}

func NewBallot(fromAddress string, round round.Round, transactions []string) (b *Ballot) {
	body := BallotBody{
		Source: fromAddress,
		Proposed: BallotBodyProposed{
			Proposer:     fromAddress,
			Round:        round,
			Transactions: transactions,
		},
		State: StateINIT,
		Vote:  VotingNOTYET,
	}

	if len(transactions) < 1 {
		body.Vote = VotingYES
	}

	b = &Ballot{
		H: BallotHeader{},
		B: body,
	}

	return
}

func NewBallotFromJSON(data []byte) (b Ballot, err error) {
	if err = json.Unmarshal(data, &b); err != nil {
		return
	}

	return
}

func (b Ballot) GetType() string {
	return common.BallotMessage
}

func (b Ballot) GetHash() string {
	return b.H.Hash
}

func (b Ballot) Serialize() (encoded []byte, err error) {
	encoded, err = json.Marshal(b)
	return
}

func (b Ballot) String() string {
	encoded, _ := json.MarshalIndent(b, "", "  ")
	return string(encoded)
}

func (b Ballot) IsWellFormed(networkID []byte) (err error) {
	if b.TransactionsLength() > common.MaxTransactionsInBallot {
		err = errors.ErrorBallotHasOverMaxTransactionsInBallot
		return
	}

	if !b.B.State.IsValid() {
		err = errors.ErrorInvalidState
		return
	}

	var confirmed, proposerConfirmed time.Time
	if confirmed, err = common.ParseISO8601(b.B.Confirmed); err != nil {
		return
	}
	if proposerConfirmed, err = common.ParseISO8601(b.ProposerConfirmed()); err != nil {
		return
	}

	now := time.Now()
	timeStart := now.Add(time.Duration(-1) * common.BallotConfirmedTimeAllowDuration)
	timeEnd := now.Add(common.BallotConfirmedTimeAllowDuration)
	if confirmed.Before(timeStart) || confirmed.After(timeEnd) {
		err = errors.ErrorMessageHasIncorrectTime
		return
	}
	if proposerConfirmed.Before(timeStart) || proposerConfirmed.After(timeEnd) {
		err = errors.ErrorMessageHasIncorrectTime
		return
	}

	if err = b.ProposerTransaction().IsWellFormedWithBallot(networkID, b); err != nil {
		return
	}

	if err = b.Verify(networkID); err != nil {
		return
	}

	return
}

func (b Ballot) Equal(m common.Message) bool {
	return b.H.Hash == m.GetHash()
}

func (b Ballot) Source() string {
	return b.B.Source
}

func (b Ballot) Round() round.Round {
	return b.B.Proposed.Round
}

func (b Ballot) Proposer() string {
	return b.B.Proposed.Proposer
}

func (b Ballot) Transactions() []string {
	return b.B.Proposed.Transactions
}

func (b Ballot) Confirmed() string {
	return b.B.Confirmed
}

func (b Ballot) ProposerConfirmed() string {
	return b.B.Proposed.Confirmed
}

func (b Ballot) Vote() VotingHole {
	return b.B.Vote
}

func (b *Ballot) SetSource(source string) {
	b.B.Source = source
}

func (b *Ballot) SetVote(state State, vote VotingHole) {
	b.B.State = state
	b.B.Vote = vote
}

func (b *Ballot) SetReason(reason *errors.Error) {
	b.B.Reason = reason
}

func (b *Ballot) TransactionsLength() int {
	return len(b.B.Proposed.Transactions)
}

func (b *Ballot) SignByProposer(kp keypair.KP, networkID []byte) {
	ptx := b.ProposerTransaction()
	ptx.Sign(kp, networkID)
	b.SetProposerTransaction(ptx)

	b.B.Proposed.Confirmed = common.NowISO8601()
	hash := common.MustMakeObjectHash(b.B.Proposed)
	signature, _ := common.MakeSignature(kp, networkID, string(hash))
	b.H.ProposerSignature = base58.Encode(signature)
}

func (b *Ballot) Sign(kp keypair.KP, networkID []byte) {
	if kp.Address() == b.B.Proposed.Proposer && b.State() == StateINIT {
		b.SignByProposer(kp, networkID)
	}

	b.B.Confirmed = common.NowISO8601()
	b.B.Source = kp.Address()
	b.H.Hash = b.B.MakeHashString()
	signature, _ := common.MakeSignature(kp, networkID, b.H.Hash)
	b.H.Signature = base58.Encode(signature)

	return
}

func (b Ballot) Verify(networkID []byte) (err error) {
	var kp keypair.KP
	if kp, err = keypair.Parse(b.B.Proposed.Proposer); err != nil {
		return
	}
	err = kp.Verify(
		append(networkID, common.MustMakeObjectHash(b.B.Proposed)...),
		base58.Decode(b.H.ProposerSignature),
	)
	if err != nil {
		return
	}

	if kp, err = keypair.Parse(b.B.Source); err != nil {
		return
	}
	err = kp.Verify(
		append(networkID, []byte(b.H.Hash)...),
		base58.Decode(b.H.Signature),
	)
	if err != nil {
		return
	}

	return
}

func (b Ballot) IsFromProposer() bool {
	return b.B.Source == b.B.Proposed.Proposer
}

func (b Ballot) State() State {
	return b.B.State
}

func (b Ballot) ProposerTransaction() ProposerTransaction {
	return b.B.Proposed.ProposerTransaction
}

// SetProposerTransaction should be set in `Ballot`, without it can not be
// passed thru `Ballot.IsWellFormed()`.
func (b *Ballot) SetProposerTransaction(ptx ProposerTransaction) {
	b.B.Proposed.ProposerTransaction = ptx
}

type BallotHeader struct {
	Hash              string `json:"hash"`               // hash of `BallotBody`
	Signature         string `json:"signature"`          // signed by source node of <networkID> + `Hash`
	ProposerSignature string `json:"proposer-signature"` // signed by proposer of <networkID> + `Hash` of `BallotBodyProposed`
}

type BallotBodyProposed struct {
	Confirmed           string              `json:"confirmed"` // created time, ISO8601
	Proposer            string              `json:"proposer"`
	Round               round.Round         `json:"round"`
	Transactions        []string            `json:"transactions"`
	ProposerTransaction ProposerTransaction `json:"proposer-transaction"`
}

type BallotBody struct {
	Confirmed string             `json:"confirmed"` // created time, ISO8601
	Proposed  BallotBodyProposed `json:"proposed"`
	Source    string             `json:"source"`
	State     State              `json:"state"`
	Vote      VotingHole         `json:"vote"`
	Reason    *errors.Error      `json:"reason"`
}

func (rb BallotBody) MakeHash() []byte {
	return common.MustMakeObjectHash(rb)
}

func (rb BallotBody) MakeHashString() string {
	return base58.Encode(rb.MakeHash())
}
