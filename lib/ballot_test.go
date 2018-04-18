package sebak

import (
	"testing"
)

func TestHashingBallot(t *testing.T) {
	ballot := Ballot{
		Hash: "this-is-hash",
		Vote: true,
	}
	if _, err := GetObjectHash(ballot); err != nil {
		t.Errorf("`Ballot` must be hashable: %v", err)
	}
}
