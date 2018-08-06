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

func (rb Ballot) GetType() string {
	return sebaknetwork.BallotMessage
}

func (rb Ballot) GetHash() string {
	return rb.H.Hash
}

func (rb Ballot) Serialize() (encoded []byte, err error) {
	encoded, err = json.Marshal(rb)
	return
}

func (rb Ballot) String() string {
	encoded, _ := json.MarshalIndent(rb, "", "  ")
	return string(encoded)
}

func (rb Ballot) IsWellFormed(networkID []byte) (err error) {
	if err = rb.Verify(networkID); err != nil {
		return
	}

	return
}

func (rb Ballot) Equal(m sebakcommon.Message) bool {
	return rb.H.Hash == m.GetHash()
}

func (rb Ballot) Source() string {
	return rb.B.Source
}

func (rb Ballot) Round() Round {
	return rb.B.Proposed.Round
}

func (rb Ballot) Proposer() string {
	return rb.B.Proposed.NewBallot
}

func (rb Ballot) Transactions() []string {
	return rb.B.Proposed.Transactions
}

func (rb Ballot) ValidTransactions() map[string]bool {
	return rb.B.Proposed.ValidTransactions
}

func (rb Ballot) ValidTransactionSlice() []string {
	slice := []string{}
	for hash := range rb.B.Proposed.ValidTransactions {
		slice = append(slice, hash)
	}
	return slice
}

func (rb Ballot) Confirmed() string {
	return rb.B.Confirmed
}

func (rb Ballot) ProposerConfirmed() string {
	return rb.B.Proposed.Confirmed
}

func (rb Ballot) Vote() VotingHole {
	return rb.B.Vote
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

func (rb *Ballot) SetSource(source string) {
	rb.B.Source = source
}

func (rb *Ballot) SetVote(vote VotingHole) {
	rb.B.Vote = vote
}

func (rb *Ballot) SetReason(reason *sebakerror.Error) {
	rb.B.Reason = reason
}

func (rb *Ballot) TransactionsLength() int {
	return len(rb.B.Proposed.Transactions)
}

func (rb *Ballot) ValidTransactionsLength() int {
	return len(rb.B.Proposed.ValidTransactions)
}

func (rb *Ballot) IsValidTransaction(hash string) bool {
	_, ok := rb.B.Proposed.ValidTransactions[hash]
	return ok
}

func (rb *Ballot) SetValidTransactions(validTransactions map[string]bool) {
	rb.B.Proposed.ValidTransactions = validTransactions
}

func (rb *Ballot) Sign(kp keypair.KP, networkID []byte) {
	if kp.Address() == rb.B.Proposed.NewBallot {
		rb.B.Proposed.Confirmed = sebakcommon.NowISO8601()
		hash := sebakcommon.MustMakeObjectHash(rb.B.Proposed)
		signature, _ := kp.Sign(append(networkID, []byte(hash)...))
		rb.H.ProposerSignature = base58.Encode(signature)
	}

	rb.B.Confirmed = sebakcommon.NowISO8601()
	rb.B.Source = kp.Address()
	rb.H.Hash = rb.B.MakeHashString()
	signature, _ := kp.Sign(append(networkID, []byte(rb.H.Hash)...))

	rb.H.Signature = base58.Encode(signature)

	return
}

func (rb Ballot) Verify(networkID []byte) (err error) {
	var kp keypair.KP
	if kp, err = keypair.Parse(rb.B.Proposed.NewBallot); err != nil {
		return
	}
	err = kp.Verify(
		append(networkID, sebakcommon.MustMakeObjectHash(rb.B.Proposed)...),
		base58.Decode(rb.H.ProposerSignature),
	)
	if err != nil {
		return
	}

	if kp, err = keypair.Parse(rb.B.Source); err != nil {
		return
	}
	err = kp.Verify(
		append(networkID, []byte(rb.H.Hash)...),
		base58.Decode(rb.H.Signature),
	)
	if err != nil {
		return
	}

	return
}

func (rb Ballot) IsFromProposer() bool {
	return rb.B.Source == rb.B.Proposed.NewBallot
}

func NewBallot(localNode *sebaknode.LocalNode, round Round, transactions []string) (rb *Ballot) {
	body := BallotBody{
		Source: localNode.Address(),
		Proposed: BallotBodyProposed{
			NewBallot:    localNode.Address(),
			Round:        round,
			Transactions: transactions,
		},
	}
	rb = &Ballot{
		H: BallotHeader{},
		B: body,
	}

	return
}

func NewBallotFromJSON(data []byte) (rb Ballot, err error) {
	if err = json.Unmarshal(data, &rb); err != nil {
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
