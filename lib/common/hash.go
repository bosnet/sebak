package common

import (
	"crypto/sha256"
	"github.com/btcsuite/btcutil/base58"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
)

func MakeHash(b []byte) []byte {
	first := sha256.Sum256(b)
	second := sha256.Sum256(first[:])
	return second[:]
}

func MakeObjectHash(i interface{}) (b []byte, err error) {
	var e []byte
	if e, err = rlp.EncodeToBytes(i); err != nil {
		return
	}

	b = MakeHash(e)

	return
}

func MustMakeObjectHash(i interface{}) []byte {
	if b, err := MakeObjectHash(i); err != nil {
		panic(err)
	} else {
		return b
	}
}

func MustMakeObjectHashString(i interface{}) string {
	b := MustMakeObjectHash(i)
	return base58.Encode(b)
}

type Hash = ethcommon.Hash

func BytesToHash(b []byte) Hash {
	var h Hash
	h.SetBytes(b)
	return h
}
