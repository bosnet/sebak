package common

import (
	"testing"
)

type intType uint64

type intSideType struct {
	I uint64
}

func TestUnsignedIntHashableRLP(t *testing.T) {
	CheckRoundTripRLP(t, uint(10))
}

func TestInt64HashableRLP(t *testing.T) {
	CheckRoundTripRLP(t, intType(10))
}

func TestInt64StructHashableRLP(t *testing.T) {
	CheckRoundTripRLP(t, intSideType{I: 64})
}
