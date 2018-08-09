package sebak

import "boscoin.io/sebak/lib/common"

const (
	// BaseFee is the default transaction fee, if fee is lower than BaseFee, the
	// transaction will fail validation.
	BaseFee sebakcommon.Amount = 10000
)
