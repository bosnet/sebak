package sebak

import (
	"testing"

	"github.com/spikeekips/sebak/lib/util"
)

func TestHashingBallot(t *testing.T) {
	ballot := Ballot{
		Hash: "this-is-hash",
		Vote: true,
	}
	if _, err := util.GetObjectHash(ballot); err != nil {
		t.Errorf("`Ballot` must be hashable: %v", err)
	}
}
