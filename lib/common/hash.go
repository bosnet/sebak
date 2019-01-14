//
// This package defines hash-related functions used in Sebak
// The primary hash function used in Sebak is a double sha256,
// like Bitcoin.
//
package common

import (
	"crypto/sha256"
	"io"

	"github.com/btcsuite/btcutil/base58"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
)

// 32 bytes / 256 bits hash type
type Hash = ethcommon.Hash

// Set the content of the hash to be the provided binary data
var BytesToHash = ethcommon.BytesToHash

// Generate a double SHA-256 hash out of the supplied binary data
func MakeHash(b []byte) []byte {
	first := sha256.Sum256(b)
	second := sha256.Sum256(first[:])
	return second[:]
}

// Interface recognized by RLP encoder
type Encoder = rlp.Encoder

// Interface recognized by RLP decoder
type Decoder = rlp.Decoder

// Argument to the `Decoder.Decode` method
type RLPStream = rlp.Stream

// Encode the provided value
// It is exposed here as it is useful for recursive calls
// by types implementing `Encoder`
var Encode = rlp.Encode

// Ditto
var EncodeToBytes = rlp.EncodeToBytes

// Ditto
var EncodeToReader = rlp.EncodeToReader

// Computes the minimum number of bytes required to store `i` in RLP encoding.
func SizeofSize(i uint64) (size byte) {
	for size = 1; ; size++ {
		if i >>= 8; i == 0 {
			return size
		}
	}
}

// Write the length of a list to the writer according to RLP specs
// See: https://github.com/ethereum/wiki/wiki/RLP
func PutListLength(w io.Writer, length uint64) error {
	if length <= 55 {
		if _, err := w.Write([]byte{byte(0xc0 + length)}); err != nil {
			return err
		}
	} else {
		if _, err := w.Write([]byte{0xf7 + SizeofSize(length)}); err != nil {
			return err
		}
		PutInt(w, length)
	}
	return nil
}

// Writes i to the `io.Writer` in big endian
func PutInt(w io.Writer, i uint64) error {
	switch {
	case i < (1 << 8):
		_, err := w.Write([]byte{byte(i)})
		return err
	case i < (1 << 16):
		_, err := w.Write([]byte{byte(i >> 8), byte(i)})
		return err
	case i < (1 << 24):
		_, err := w.Write([]byte{byte(i >> 16), byte(i >> 8), byte(i)})
		return err
	case i < (1 << 32):
		_, err := w.Write([]byte{
			byte(i >> 24), byte(i >> 16), byte(i >> 8), byte(i)})
		return err
	case i < (1 << 40):
		_, err := w.Write([]byte{
			byte(i >> 32), byte(i >> 24), byte(i >> 16), byte(i >> 8),
			byte(i)})
		return err
	case i < (1 << 48):
		_, err := w.Write([]byte{
			byte(i >> 40), byte(i >> 32), byte(i >> 24), byte(i >> 16),
			byte(i >> 8), byte(i)})
		return err
	case i < (1 << 56):
		_, err := w.Write([]byte{
			byte(i >> 48), byte(i >> 40), byte(i >> 32), byte(i >> 24),
			byte(i >> 16), byte(i >> 8), byte(i)})
		return err
	default:
		_, err := w.Write([]byte{
			byte(i >> 56), byte(i >> 48), byte(i >> 40), byte(i >> 32),
			byte(i >> 24), byte(i >> 16), byte(i >> 8), byte(i)})
		return err
	}
}

// Generate a double SHA-256 hash from the interface's RLP encoding
//
// Types wishing to implement custom encoding should implement
// the `Encoder.EncodeRLP` interface
func MakeObjectHash(i interface{}) (b []byte, err error) {
	var e []byte
	if e, err = rlp.EncodeToBytes(i); err != nil {
		return
	}

	b = MakeHash(e)

	return
}

// Pedestrian version of `MakeObjectHash`
func MustMakeObjectHash(i interface{}) []byte {
	if b, err := MakeObjectHash(i); err != nil {
		panic(err)
	} else {
		return b
	}
}

// Returns the hash of the object, base58 encoded
func MustMakeObjectHashString(i interface{}) string {
	b := MustMakeObjectHash(i)
	return base58.Encode(b)
}
