package block

import (
	"time"

	"boscoin.io/sebak/lib/consensus/round"
)

type Header struct {
	Version          uint32    `json:"Version"`
	PrevBlockHash    string    `json:"prev-block-hash"`   // TODO Uint256 type
	TransactionsRoot string    `json:"transactions-root"` // Merkle root of Txs // TODO Uint256 type
	Timestamp        time.Time `json:"timestamp"`
	Height           uint64    `json:"height"`
	TotalTxs         uint64    `json:"total-txs"`

	// TODO smart contract fields
}

func NewBlockHeader(round round.Round, currentTxs uint64, txRoot string) *Header {
	return &Header{
		PrevBlockHash:    round.BlockHash,
		Timestamp:        time.Now(),
		Height:           round.BlockHeight + 1,
		TotalTxs:         round.TotalTxs + currentTxs,
		TransactionsRoot: txRoot,
	}
}
