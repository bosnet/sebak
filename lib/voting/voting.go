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
	SetValidators(int) error
	Connected() int
	SetConnected(int) error
}
