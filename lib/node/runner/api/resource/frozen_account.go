package resource

import (
	"strings"

	"boscoin.io/sebak/lib/block"
	"boscoin.io/sebak/lib/common"
	"github.com/nvellon/hal"
)

type FrozenAccountState string

const (
	FrozenState   FrozenAccountState = "frozen"
	MeltingState  FrozenAccountState = "melting"
	UnfrozenState FrozenAccountState = "unfrozen"
	ReturnedState FrozenAccountState = "returned"
)

type FrozenAccount struct {
	ba   *block.BlockAccount
	info FrozenAccountInfo
}

func NewFrozenAccount(ba *block.BlockAccount, info FrozenAccountInfo) *FrozenAccount {
	fa := &FrozenAccount{
		ba:   ba,
		info: info,
	}
	return fa
}

type FrozenAccountInfo struct {
	CreatedBlockHeight           uint64
	CreatedOpHash                string
	CreatedSequenceId            uint64
	InitialAmount                common.Amount
	FreezingState                FrozenAccountState
	UnfreezingRequestBlockHeight uint64
	UnfreezingRequestOpHash      string
	UnfreezingRemainingBlocks    uint64
	PaymentOpHash                string
}

func (fa FrozenAccount) GetMap() hal.Entry {
	return hal.Entry{
		"address":                     fa.ba.Address,
		"linked":                      fa.ba.Linked,
		"create_block_height":         fa.info.CreatedBlockHeight,
		"create_op_hash":              fa.info.CreatedOpHash,
		"sequence_id":                 fa.info.CreatedSequenceId,
		"amount":                      fa.info.InitialAmount,
		"state":                       fa.info.FreezingState,
		"unfreezing_block_height":     fa.info.UnfreezingRequestBlockHeight,
		"unfreezing_op_hash":          fa.info.UnfreezingRequestOpHash,
		"unfreezing_remaining_blocks": fa.info.UnfreezingRemainingBlocks,
		"payment_op_hash":             fa.info.PaymentOpHash,
	}
}

func (fa FrozenAccount) Resource() *hal.Resource {
	r := hal.NewResource(fa, fa.LinkSelf())

	return r
}

func (fa FrozenAccount) LinkSelf() string {
	address := fa.ba.Linked

	return strings.Replace(URLAccountFrozenAccounts, "{id}", address, -1)
}

func (fa FrozenAccount) MarshalJSON() ([]byte, error) {
	r := fa.Resource()
	return common.JSONMarshalWithoutEscapeHTML(r.GetMap())
}
