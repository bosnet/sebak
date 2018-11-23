package voting

type Hole string

const (
	NOTYET Hole = "NOT-YET"
	YES    Hole = "YES"
	NO     Hole = "NO"
	EXP    Hole = "EXPIRED"
)

type ThresholdPolicy interface {
	Threshold() int
	Validators() int
	// Set the number of validators required for consensus
	// The parameter must be a strictly positive integer
	SetValidators(int)
	Connected() int
	// Set the number of currently connected nodes
	// The parameter must be a strictly positive integer
	SetConnected(int)
}
