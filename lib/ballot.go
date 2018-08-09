package sebak

import (
	"encoding/json"

	"github.com/btcsuite/btcutil/base58"
	"github.com/stellar/go/keypair"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/error"
	"boscoin.io/sebak/lib/network"
	"boscoin.io/sebak/lib/node"
	"boscoin.io/sebak/lib/storage"
)

type Ballot struct {
	H BallotHeader
	B BallotBody
}

func (b Ballot) GetType() string {
	return sebaknetwork.BallotMessage
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

func (b Ballot) Round() Round {
	return b.B.Proposed.Round
}

func (b Ballot) Proposer() string {
	return b.B.Proposed.NewBallot
}

func (b Ballot) Transactions() []string {
	return b.B.Proposed.Transactions
}

func (b Ballot) ValidTransactions() map[string]bool {
	return b.B.Proposed.ValidTransactions
}

func (b Ballot) ValidTransactionSlice() []string {
	slice := []string{}
	for hash := range b.B.Proposed.ValidTransactions {
		slice = append(slice, hash)
	}
	return slice
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

type BallotHeader struct {
	Hash              string `json:"hash"`               // hash of `BallotBody`
	Signature         string `json:"signature"`          // signed by source node of <networkID> + `Hash`
	ProposerSignature string `json:"proposer-signature"` // signed by proposer of <networkID> + `Hash` of `BallotBodyProposed`
}

type BallotBodyProposed struct {
	Confirmed         string          `json:"confirmed"` // created time, ISO8601
	NewBallot         string          `json:"proposer"`
	Round             Round           `json:"round"`
	Transactions      []string        `json:"transactions"`
	ValidTransactions map[string]bool `json:"valid-transactions"`
}

type BallotBody struct {
	Confirmed string             `json:"confirmed"` // created time, ISO8601
	Proposed  BallotBodyProposed `json:"proposed"`
	Source    string             `json:"source"`

	Vote   VotingHole        `json:"vote"`
	Reason *sebakerror.Error `json:"reason"`
}

func (rbody BallotBody) MakeHash() []byte {
	return sebakcommon.MustMakeObjectHash(rbody)
}

func (rbody BallotBody) MakeHashString() string {
	return base58.Encode(rbody.MakeHash())
}

func (b *Ballot) SetSource(source string) {
	b.B.Source = source
}

func (b *Ballot) SetVote(vote VotingHole) {
	b.B.Vote = vote
}

func (b *Ballot) SetReason(reason *sebakerror.Error) {
	b.B.Reason = reason
}

func (b *Ballot) TransactionsLength() int {
	return len(b.B.Proposed.Transactions)
}

func (b *Ballot) ValidTransactionsLength() int {
	return len(b.B.Proposed.ValidTransactions)
}

func (b *Ballot) IsValidTransaction(hash string) bool {
	_, ok := b.B.Proposed.ValidTransactions[hash]
	return ok
}

func (b *Ballot) SetValidTransactions(validTransactions map[string]bool) {
	b.B.Proposed.ValidTransactions = validTransactions
}

func (b *Ballot) Sign(kp keypair.KP, networkID []byte) {
	if kp.Address() == b.B.Proposed.NewBallot {
		b.B.Proposed.Confirmed = sebakcommon.NowISO8601()
		hash := sebakcommon.MustMakeObjectHash(b.B.Proposed)
		signature, _ := kp.Sign(append(networkID, []byte(hash)...))
		b.H.ProposerSignature = base58.Encode(signature)
	}

	b.B.Confirmed = sebakcommon.NowISO8601()
	b.B.Source = kp.Address()
	b.H.Hash = b.B.MakeHashString()
	signature, _ := kp.Sign(append(networkID, []byte(b.H.Hash)...))

	b.H.Signature = base58.Encode(signature)

	return
}

func (b Ballot) Verify(networkID []byte) (err error) {
	var kp keypair.KP
	if kp, err = keypair.Parse(b.B.Proposed.NewBallot); err != nil {
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
	return b.B.Source == b.B.Proposed.NewBallot
}

func NewBallot(localNode *sebaknode.LocalNode, round Round, transactions []string) (b *Ballot) {
	body := BallotBody{
		Source: localNode.Address(),
		Proposed: BallotBodyProposed{
			NewBallot:    localNode.Address(),
			Round:        round,
			Transactions: transactions,
		},
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

func FinishBallot(st *sebakstorage.LevelDBBackend, ballot Ballot, transactionPool *TransactionPool) (block Block, err error) {
	var ts *sebakstorage.LevelDBBackend
	if ts, err = st.OpenTransaction(); err != nil {
		return
	}

	transactions := map[string]Transaction{}
	for hash, _ := range ballot.B.Proposed.ValidTransactions {
		tx, found := transactionPool.Get(hash)
		if !found {
			err = sebakerror.ErrorTransactionNotFound
			return
		}
		transactions[hash] = tx
	}

	for hash, _ := range ballot.B.Proposed.ValidTransactions {
		tx := transactions[hash]
		raw, _ := json.Marshal(tx)

		bt := NewBlockTransactionFromTransaction(tx, raw)
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

		var baSource *BlockAccount
		if baSource, err = GetBlockAccount(ts, tx.B.Source); err != nil {
			err = sebakerror.ErrorBlockAccountDoesNotExists
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

	block = NewBlockFromBallot(ballot)
	if err = block.Save(ts); err != nil {
		return
	}

	if err = ts.Commit(); err != nil {
		ts.Discard()
	}

	return
}
