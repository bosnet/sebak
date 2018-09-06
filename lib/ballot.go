package sebak

import (
	"encoding/json"
	"time"

	"github.com/btcsuite/btcutil/base58"
	"github.com/stellar/go/keypair"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/error"
	"boscoin.io/sebak/lib/network"
	"boscoin.io/sebak/lib/node"
	"boscoin.io/sebak/lib/round"
	"boscoin.io/sebak/lib/storage"
)

type Ballot struct {
	H BallotHeader
	B BallotBody
}

func NewBallot(localNode *sebaknode.LocalNode, round round.Round, transactions []string) (b *Ballot) {
	body := BallotBody{
		Source: localNode.Address(),
		Proposed: BallotBodyProposed{
			Proposer:     localNode.Address(),
			Round:        round,
			Transactions: transactions,
		},
		State: sebakcommon.BallotStateINIT,
		Vote:  sebakcommon.VotingNOTYET,
	}

	if len(transactions) < 1 {
		body.Vote = sebakcommon.VotingYES
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
	return network.BallotMessage
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
	if b.TransactionsLength() > MaxTransactionsInBallot {
		err = errors.ErrorBallotHasOverMaxTransactionsInBallot
		return
	}

	if !b.B.State.IsValid() {
		err = errors.ErrorInvalidState
		return
	}

	var confirmed, proposerConfirmed time.Time
	if confirmed, err = sebakcommon.ParseISO8601(b.B.Confirmed); err != nil {
		return
	}
	if proposerConfirmed, err = sebakcommon.ParseISO8601(b.ProposerConfirmed()); err != nil {
		return
	}

	now := time.Now()
	timeStart := now.Add(time.Duration(-1) * BallotConfirmedTimeAllowDuration)
	timeEnd := now.Add(BallotConfirmedTimeAllowDuration)
	if confirmed.Before(timeStart) || confirmed.After(timeEnd) {
		err = errors.ErrorMessageHasIncorrectTime
		return
	}
	if proposerConfirmed.Before(timeStart) || proposerConfirmed.After(timeEnd) {
		err = errors.ErrorMessageHasIncorrectTime
		return
	}

	if err = b.Verify(networkID); err != nil {
		return
	}

	return
}

func (b Ballot) Equal(m sebakcommon.Message) bool {
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

func (b Ballot) Vote() sebakcommon.VotingHole {
	return b.B.Vote
}

func (b *Ballot) SetSource(source string) {
	b.B.Source = source
}

func (b *Ballot) SetVote(state sebakcommon.BallotState, vote sebakcommon.VotingHole) {
	b.B.State = state
	b.B.Vote = vote
}

func (b *Ballot) SetReason(reason *errors.Error) {
	b.B.Reason = reason
}

func (b *Ballot) TransactionsLength() int {
	return len(b.B.Proposed.Transactions)
}

func (b *Ballot) Sign(kp keypair.KP, networkID []byte) {
	if kp.Address() == b.B.Proposed.Proposer {
		b.B.Proposed.Confirmed = sebakcommon.NowISO8601()
		hash := sebakcommon.MustMakeObjectHash(b.B.Proposed)
		signature, _ := sebakcommon.MakeSignature(kp, networkID, string(hash))
		b.H.ProposerSignature = base58.Encode(signature)
	}

	b.B.Confirmed = sebakcommon.NowISO8601()
	b.B.Source = kp.Address()
	b.H.Hash = b.B.MakeHashString()
	signature, _ := sebakcommon.MakeSignature(kp, networkID, b.H.Hash)
	b.H.Signature = base58.Encode(signature)

	return
}

func (b Ballot) Verify(networkID []byte) (err error) {
	var kp keypair.KP
	if kp, err = keypair.Parse(b.B.Proposed.Proposer); err != nil {
		return
	}
	err = kp.Verify(
		append(networkID, sebakcommon.MustMakeObjectHash(b.B.Proposed)...),
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

func (b Ballot) State() sebakcommon.BallotState {
	return b.B.State
}

type BallotHeader struct {
	Hash              string `json:"hash"`               // hash of `BallotBody`
	Signature         string `json:"signature"`          // signed by source node of <networkID> + `Hash`
	ProposerSignature string `json:"proposer-signature"` // signed by proposer of <networkID> + `Hash` of `BallotBodyProposed`
}

type BallotBodyProposed struct {
	Confirmed    string      `json:"confirmed"` // created time, ISO8601
	Proposer     string      `json:"proposer"`
	Round        round.Round `json:"round"`
	Transactions []string    `json:"transactions"`
}

type BallotBody struct {
	Confirmed string                  `json:"confirmed"` // created time, ISO8601
	Proposed  BallotBodyProposed      `json:"proposed"`
	Source    string                  `json:"source"`
	State     sebakcommon.BallotState `json:"state"`
	Vote      sebakcommon.VotingHole  `json:"vote"`
	Reason    *errors.Error       `json:"reason"`
}

func (rb BallotBody) MakeHash() []byte {
	return sebakcommon.MustMakeObjectHash(rb)
}

func (rb BallotBody) MakeHashString() string {
	return base58.Encode(rb.MakeHash())
}

func FinishBallot(st *sebakstorage.LevelDBBackend, ballot Ballot, transactionPool *TransactionPool) (blk Block, err error) {
	var ts *sebakstorage.LevelDBBackend
	if ts, err = st.OpenTransaction(); err != nil {
		return
	}

	transactions := map[string]Transaction{}
	for _, hash := range ballot.B.Proposed.Transactions {
		tx, found := transactionPool.Get(hash)
		if !found {
			err = errors.ErrorTransactionNotFound
			return
		}
		transactions[hash] = tx
	}

	blk = NewBlockFromBallot(ballot)
	if err = blk.Save(ts); err != nil {
		return
	}

	for _, hash := range ballot.B.Proposed.Transactions {
		tx := transactions[hash]
		raw, _ := json.Marshal(tx)

		bt := NewBlockTransactionFromTransaction(blk.Hash, tx, raw)
		if err = bt.Save(ts); err != nil {
			ts.Discard()
			return
		}
		for _, op := range tx.B.Operations {
			if err = FinishOperation(ts, tx, op); err != nil {
				ts.Discard()
				return
			}
		}

		var baSource *block.BlockAccount
		if baSource, err = block.GetBlockAccount(ts, tx.B.Source); err != nil {
			err = errors.ErrorBlockAccountDoesNotExists
			ts.Discard()
			return
		}

		if err = baSource.Withdraw(tx.TotalAmount(true), tx.NextSourceCheckpoint()); err != nil {
			ts.Discard()
			return
		}

		if err = baSource.Save(ts); err != nil {
			ts.Discard()
			return
		}

	}

	if err = ts.Commit(); err != nil {
		ts.Discard()
	}

	return
}
