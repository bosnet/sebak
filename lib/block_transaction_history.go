package sebak

import "github.com/spikeekips/sebak/lib/util"

/*
BlockTransactionHistory is for keeping `Transaction` history. the storage should support,
 * find by `Hash`
 * find by `Source`
 * sort by `Confirmed`
 * sort by `Created`
*/
type BlockTransactionHistory struct {
	Hash   string
	Source string

	Confirmed string
	Created   string
	Message   string
}

func NewTransactionHistoryFromTransaction(tx Transaction, message string) BlockTransactionHistory {
	return BlockTransactionHistory{
		Hash:      tx.H.Hash,
		Source:    tx.B.Source,
		Confirmed: util.NowISO8601(),
		Created:   tx.H.Created,
		Message:   message,
	}
}

/*
BlockTransactionError stores all the non-confirmed transactions and it's reason.
the storage should support,
 * find by `Hash`
*/
type BlockTransactionError struct {
	Hash string

	Reason string
}
