package block

import (
	"encoding/json"
	"time"

	"boscoin.io/sebak/lib/consensus/round"
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

func NewBlockHeader(round round.Round, txRoot string) *Header {
	return &Header{
		PrevBlockHash:    round.BlockHash,
		Timestamp:        time.Now(),
		Height:           round.BlockHeight + 1,
		TotalTxs:         round.TotalTxs,
		TotalOps:         round.TotalOps,
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
