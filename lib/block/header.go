package block

import (
	"encoding/json"
	"time"

	"boscoin.io/sebak/lib/voting"
)

type Header struct {
	// TODO rename `Header` to `BlockHeader`
	Version          uint32    `json:"version"`
	PrevBlockHash    string    `json:"prev_block_hash"`   // TODO Uint256 type
	TransactionsRoot string    `json:"transactions_root"` // Merkle root of Txs // TODO Uint256 type
	Timestamp        time.Time `json:"timestamp"`
	Height           uint64    `json:"height"`
	TotalTxs         uint64    `json:"total-txs"`
	TotalOps         uint64    `json:"total-ops"`

	// TODO smart contract fields
}

func NewBlockHeader(basis voting.Basis, txRoot string) *Header {
	return &Header{
		PrevBlockHash:    basis.BlockHash,
		Timestamp:        time.Now(),
		Height:           basis.Height,
		TotalTxs:         basis.TotalTxs,
		TotalOps:         basis.TotalOps,
		TransactionsRoot: txRoot,
	}
}

func (h Header) Serialize() (encoded []byte, err error) {
	encoded, err = json.Marshal(h)
	return
}

func (h Header) String() string {
	encoded, _ := json.MarshalIndent(h, "", "  ")
	return string(encoded)
}
