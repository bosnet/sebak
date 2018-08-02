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

type Round struct {
	Number      uint64 `json:"number"`       // round sequence number
	BlockHeight uint64 `json:"block-height"` // last block height
	BlockHash   string `json:"block-hash"`   // hash of last block
	TotalTxs    uint64 `json:"total-txs"`
}

func (r Round) Hash() string {
	return base58.Encode(sebakcommon.MustMakeObjectHash(r))
}

func (r Round) IsSame(a Round) bool {
	return r.Hash() == a.Hash()
}

type RoundBallot struct {
	H RoundBallotHeader
	B RoundBallotBody
}

func (rb RoundBallot) GetType() string {
	return sebaknetwork.RoundBallotMessage
}

func (rb RoundBallot) GetHash() string {
	return rb.H.Hash
}

func (rb RoundBallot) Serialize() (encoded []byte, err error) {
	encoded, err = json.Marshal(rb)
	return
}

func (rb RoundBallot) String() string {
	encoded, _ := json.MarshalIndent(rb, "", "  ")
	return string(encoded)
}

func (rb RoundBallot) IsWellFormed(networkID []byte) (err error) {
	if err = rb.Verify(networkID); err != nil {
		return
	}

	return
}

func (rb RoundBallot) Equal(m sebakcommon.Message) bool {
	return rb.H.Hash == m.GetHash()
}

func (rb RoundBallot) Source() string {
	return rb.B.Source
}

func (rb RoundBallot) Round() Round {
	return rb.B.Proposed.Round
}

func (rb RoundBallot) Proposer() string {
	return rb.B.Proposed.Proposer
}

func (rb RoundBallot) Transactions() []string {
	return rb.B.Proposed.Transactions
}

func (rb RoundBallot) ValidTransactions() []string {
	return rb.B.Proposed.ValidTransactions
}

func (rb RoundBallot) Confirmed() string {
	return rb.B.Confirmed
}

func (rb RoundBallot) ProposerConfirmed() string {
	return rb.B.Proposed.Confirmed
}

func (rb RoundBallot) Vote() VotingHole {
	return rb.B.Vote
}

type RoundBallotHeader struct {
	Hash              string `json:"hash"`               // hash of `RoundBallotBody`
	Signature         string `json:"signature"`          // signed by source node of <networkID> + `Hash`
	ProposerSignature string `json:"proposer-signature"` // signed by proposer of <networkID> + `Hash` of `RoundBallotBodyProposed`
}

type RoundBallotBodyProposed struct {
	Confirmed         string   `json:"confirmed"` // created time, ISO8601
	Proposer          string   `json:"proposer"`
	Round             Round    `json:"round"`
	Transactions      []string `json:"transactions"`
	ValidTransactions []string `json:"valid-transactions"`
}

type RoundBallotBody struct {
	Confirmed string                  `json:"confirmed"` // created time, ISO8601
	Proposed  RoundBallotBodyProposed `json:"proposed"`
	Source    string                  `json:"source"`

	Vote   VotingHole        `json:"vote"`
	Reason *sebakerror.Error `json:"reason"`
}

func (rbody RoundBallotBody) MakeHash() []byte {
	return sebakcommon.MustMakeObjectHash(rbody)
}

func (rbody RoundBallotBody) MakeHashString() string {
	return base58.Encode(rbody.MakeHash())
}

func (rb *RoundBallot) SetSource(source string) {
	rb.B.Source = source
}

func (rb *RoundBallot) SetVote(vote VotingHole) {
	rb.B.Vote = vote
}

func (rb *RoundBallot) SetReason(reason *sebakerror.Error) {
	rb.B.Reason = reason
}

func (rb *RoundBallot) ValidTransactionsLength() int {
	return len(rb.B.Proposed.ValidTransactions)
}

func (rb *RoundBallot) SetValidTransactions(validTransactions []string) {
	rb.B.Proposed.ValidTransactions = validTransactions
}

func (rb *RoundBallot) Sign(kp keypair.KP, networkID []byte) {
	if kp.Address() == rb.B.Proposed.Proposer {
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

func (rb RoundBallot) Verify(networkID []byte) (err error) {
	var kp keypair.KP
	if kp, err = keypair.Parse(rb.B.Proposed.Proposer); err != nil {
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

func (rb RoundBallot) IsFromProposer() bool {
	return rb.B.Source == rb.B.Proposed.Proposer
}

func NewRoundBallot(localNode *sebaknode.LocalNode, round Round, transactions []string) (rb *RoundBallot) {
	body := RoundBallotBody{
		Source: localNode.Address(),
		Proposed: RoundBallotBodyProposed{
			Proposer:     localNode.Address(),
			Round:        round,
			Transactions: transactions,
		},
	}
	rb = &RoundBallot{
		H: RoundBallotHeader{},
		B: body,
	}

	return
}

func NewRoundBallotFromJSON(data []byte) (rb RoundBallot, err error) {
	if err = json.Unmarshal(data, &rb); err != nil {
		return
	}

	return
}

func FinishRoundBallot(st *sebakstorage.LevelDBBackend, ballot RoundBallot, transactions map[string]Transaction) (block Block, err error) {
	var ts *sebakstorage.LevelDBBackend
	if ts, err = st.OpenTransaction(); err != nil {
		return
	}

	for _, txHash := range ballot.B.Proposed.ValidTransactions {
		tx := transactions[txHash]
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

	block = NewBlockFromRoundBallot(ballot)
	if err = block.Save(ts); err != nil {
		return
	}

	if err = ts.Commit(); err != nil {
		ts.Discard()
	}

	return
}
