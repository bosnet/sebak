package trie

import "github.com/ethereum/go-ethereum/rlp"

func DecodeBytes(b []byte, val interface{}) error {
	return rlp.DecodeBytes(b, val)
}

func EncodeToBytes(val interface{}) ([]byte, error) {
	return rlp.EncodeToBytes(val)
}

func Split(b []byte) (content []byte, err error) {
	_, content, _, err = rlp.Split(b)
	return content, err
}
