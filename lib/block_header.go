package sebak

import (
	"time"
)

type Header struct {
	Version          uint32
	PrevBlockHash    string // [TODO] Uint256 type
	TransactionsRoot string // Merkle root of Txs [TODO] Uint256 type
	Timestamp        time.Time
	Height           uint64
	TotalTxs         uint64
	// prevConsensusResultHash string // [TODO] Uint256 type
	// ConsensusPayloadHash    Uint256
	// ConsensusPayload        Payload  // or []byte
	// StateRoot types.Hash    // MPT of state
	// TODO + smart contract fields
}

func NewBlockHeader(height uint64, prevBlockHash string, prevTotalTxs uint64, currentTxs uint64, txRoot string) *Header {
	return &Header{
		PrevBlockHash:    prevBlockHash,
		Timestamp:        time.Now(),
		Height:           height,
		TotalTxs:         prevTotalTxs + currentTxs,
		TransactionsRoot: txRoot,
	}
	//p.fill()
	//return &p
}

// func (h *Header) fill() {
// 	// [TODO] fill
// }

// type ConsensusResult struct {
// 	BlockHash string // [TODO] Uint256 type
// 	Ballots   []*Ballot
// }
