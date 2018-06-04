package sebakcommon

import (
	"testing"

	"github.com/ethereum/go-ethereum/rlp"
)

type intType uint64

type intSideType struct {
	i int64
}

func TestInt64HashableRLP(t *testing.T) {
	i := intType(10)
	_, err := rlp.EncodeToBytes(i)
	if err != nil {
		t.Error(err)
		return
	}
}

func TestInt64StructHashableRLP(t *testing.T) {
	i := intSideType{i: 64}
	_, err := rlp.EncodeToBytes(i)
	if err != nil {
		t.Error(err)
		return
	}
}
