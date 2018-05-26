package sebakcommon

import (
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
