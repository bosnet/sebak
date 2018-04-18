package sebak

import (
	"github.com/ethereum/go-ethereum/rlp"
	"golang.org/x/crypto/argon2"
)

var HashSalt = []byte("sebak")

func GetHash(b []byte) []byte {
	return argon2.Key(b, HashSalt, 3, 32*1024, 4, 32)
}

func GetObjectHash(i interface{}) (b []byte, err error) {
	var e []byte
	if e, err = rlp.EncodeToBytes(i); err != nil {
		return
	}

	b = argon2.Key(e, HashSalt, 3, 32*1024, 4, 32)

	return
}
