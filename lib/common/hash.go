package common

import (
	"github.com/btcsuite/btcutil/base58"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
	"golang.org/x/crypto/argon2"
)

var HashSalt = []byte("sebak")

func MakeHash(b []byte) []byte {
	return argon2.Key(b, HashSalt, 3, 32*1024, 4, 32)
}

func MakeObjectHash(i interface{}) (b []byte, err error) {
	var e []byte
	if e, err = rlp.EncodeToBytes(i); err != nil {
		return
	}

	b = MakeHash(e)

	return
}

func MustMakeObjectHash(i interface{}) (b []byte) {
	b, _ = MakeObjectHash(i)
	return
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
