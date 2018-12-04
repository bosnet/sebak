//
// This package defines hash-related functions used in Sebak
// The primary hash function used in Sebak is a double sha256,
// like Bitcoin.
//
package common

import (
	"crypto/sha256"
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
