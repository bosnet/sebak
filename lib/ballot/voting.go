package ballot

type VotingHole string

const (
	VotingNOTYET VotingHole = "NOT-YET"
	VotingYES    VotingHole = "YES"
	VotingNO     VotingHole = "NO"
	VotingEXP    VotingHole = "EXPIRED"
)

type VotingThresholdPolicy interface {
	Threshold(State) int
	Validators() int
	SetValidators(int) error
	Connected() int
	SetConnected(int) error
}
