package sebak

import (
	"time"
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

func NewBlockHeader(height uint64, prevBlockHash string, prevTotalTxs uint64, currentTxs uint64, txRoot string) *Header {
	return &Header{
		PrevBlockHash:    prevBlockHash,
		Timestamp:        time.Now(),
		Height:           height,
		TotalTxs:         prevTotalTxs + currentTxs,
		TransactionsRoot: txRoot,
	}
}
