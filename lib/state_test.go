package sebak

import (
	"testing"
)

func TestBallotStateNext(t *testing.T) {
	if BallotStateNONE.Next() != BallotStateINIT {
		t.Error("next state must be `BallotStateINIT`")
	}
	if BallotStateINIT.Next() != BallotStateSIGN {
		t.Error("next state must be `BallotStateSIGN`")
	}
	if BallotStateSIGN.Next() != BallotStateACCEPT {
		t.Error("next state must be `BallotStateACCEPT`")
	}
	if BallotStateACCEPT.Next() != BallotStateALLCONFIRM {
		t.Error("next state must be `BallotStateALLCONFIRM`")
	}
	if BallotStateALLCONFIRM.Next() != BallotStateNONE {
		t.Error("next state must be `BallotStateNONE`")
	}
}
