package operation

import (
	"boscoin.io/sebak/lib/common"
)

type UnfreezeRequest struct{}

func NewUnfreezeRequest() UnfreezeRequest {
	return UnfreezeRequest{}
}

func (o UnfreezeRequest) IsWellFormed(common.Config) (err error) {
	return
}

func (o UnfreezeRequest) HasFee() bool {
	return false
}
