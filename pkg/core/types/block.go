package types

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
)

type Uint256 [32]byte

func (o *Uint256) Bytes() []byte {
	return nil
}

func (o *Uint256) FromBytes(b []byte) {
	copy(o[32-len(b):], b)
}

type Block struct {
	Header *BlockHeader

	Transactions []*Transaction

	LastCommit *LastCommit
}

type BlockHeader struct {
	Version uint32

	PrevHeaderHash Uint256

	MerkleRoot Uint256

	Timestamp uint64

	LastCommitHash Uint256
}

type LastCommit struct {
}

type BlockBody struct {
	Transactions []*Transaction
}

func NewBlock(header *BlockHeader, txs []*Transaction) *Block {
	block := &Block{
		Header:       header,
		Transactions: txs,
	}
	return block
}

func (o *BlockHeader) Hash() Uint256 {
	var b bytes.Buffer
	binary.Write(&b, binary.BigEndian, o)
	return Uint256(sha256.Sum256(b.Bytes()))
}
