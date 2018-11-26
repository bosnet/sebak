package operation

import (
	"strconv"
	"strings"

	"boscoin.io/sebak/lib/common"
	"boscoin.io/sebak/lib/errors"
)

type InflationPF struct {
	FundingAddress string        `json:"funding_address"`
	Amount         common.Amount `json:"amount"`
	VotingResult   string        `json:"voting-result"`
}

func NewInflationPF(
	fundingAddress string,
	amount common.Amount,
	votingResult string,
) InflationPF {
	return InflationPF{
		FundingAddress: fundingAddress,
		Amount:         amount,
		VotingResult:   votingResult,
	}
}

func (o InflationPF) IsWellFormed(common.Config) (err error) {
	if int64(o.Amount) < 1 {
		err = errors.OperationAmountUnderflow
		return
	}
	if len(o.VotingResult) == 0 {
		err = errors.InvalidOperation
		return
	}

	parsedCongressVotingHash := strings.Split(o.VotingResult, "-") //0:TxHash, 1:Index
	if len(parsedCongressVotingHash) != 2 {
		return errors.InvalidOperation
	}
	if _, err = strconv.Atoi(parsedCongressVotingHash[1]); err != nil {
		return errors.InvalidOperation.Clone().SetData("error", err)
	}

	return nil
}

func (o InflationPF) GetAmount() common.Amount {
	return o.Amount
}

func (o InflationPF) HasFee() bool {
	return false
}
