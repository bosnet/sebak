package sebakcommon

type VotingThresholdPolicy interface {
	Threshold() int
	Validators() int
	SetValidators(int) error
	Connected() int
	SetConnected(int) error

	Reset(int) error
	String() string
}
