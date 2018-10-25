package client

import (
	"encoding/json"
)

type Problem struct {
	Type     string                     `json:"type"`
	Title    string                     `json:"title"`
	Status   int                        `json:"status"`
	Detail   string                     `json:"detail,omitempty"`
	Instance string                     `json:"instance,omitempty"`
	Extras   map[string]json.RawMessage `json:"extras,omitempty"`
}

type Account struct {
	Links struct {
		Self         Link `json:"self"`
		Transactions Link `json:"transactions"`
		Operations   Link `json:"operations"`
	} `json:"_links"`

	Address    string `json:"address"`
	SequenceID uint64 `json:"sequence_id"`
	Balance    string `json:"balance"`
	Linked     string `json:"linked"`
}

type Link struct {
	Href      string `json:"href"`
	Templated bool   `json:"templated,omitempty"`
}

type Transaction struct {
	Links struct {
		Self       Link `json:"self"`
		Account    Link `json:"account"`
		Operations Link `json:"operations"`
	} `json:"_links"`
	Hash           string `json:"hash"`
	Source         string `json:"source"`
	Fee            string `json:"fee"`
	SequenceID     uint64 `json:"sequence_id"`
	Created        string `json:"created"`
	OperationCount uint64 `json:"operation_count"`
}

type TransactionPost struct {
	Links struct {
		Self    Link `json:"self"`
		History Link `json:"history"`
	} `json:"_links"`
	Hash    string      `json:"hash"`
	Status  string      `json:"status"`
	Message interface{} `json:"message"`
}

type TransactionHistory struct {
	Links struct {
		Self        Link `json:"self"`
		Account     Link `json:"account"`
		Transaction Link `json:"transaction"`
	} `json:"_links"`
	Hash    string `json:"hash"`
	Source  string `json:"source"`
	Time    string `json:"time"`
	Message string `json:"message"`
	Created string `json:"created"`
	Status  string `json:"status"`
}

type TransactionsPage struct {
	Links struct {
		Self Link `json:"self"`
		Next Link `json:"next"`
		Prev Link `json:"prev"`
	} `json:"_links"`
	Embedded struct {
		Records []Transaction `json:"records"`
	} `json:"_embedded"`
}

type Operation struct {
	Links struct {
		Self        Link `json:"self"`
		Transaction Link `json:"transaction"`
	} `json:"_links"`
	Hash   string      `json:"hash"`
	Source string      `json:"source"`
	Type   string      `json:"type"`
	Body   interface{} `json:"body"`
}

type OperationsPage struct {
	Links struct {
		Self Link `json:"self"`
		Next Link `json:"next"`
		Prev Link `json:"prev"`
	} `json:"_links"`
	Embedded struct {
		Records []Operation `json:"records"`
	} `json:"_embedded"`
}

type CongressVoting struct {
	Contract []byte `json:"contract"`
	Voting   struct {
		Start uint64 `json:"start"`
		End   uint64 `json:"end"`
	} `json:"voting"`
}

type CongressVotingResult struct {
	BallotStamps struct {
		Hash string   `json:"hash"`
		Urls []string `json:"urls"`
	} `json:"ballot_stamps"`
	Voters struct {
		Hash string   `json:"hash"`
		Urls []string `json:"urls"`
	} `json:"voters"`
	Result struct {
		Count uint64 `json:"count"`
		Yes   uint64 `json:"yes"`
		No    uint64 `json:"no"`
		ABS   uint64 `json:"abs"`
	} `json:"result"`
}

type CreateAccount struct {
	Target string `json:"target"`
	Amount []byte `json:"amount"`
	Linked string `json:"linked,omitempty"`
}

type Payment struct {
	Target string `json:"target"`
	Amount []byte `json:"amount"`
}

type Inflation struct {
	Target         string `json:"target"`
	Amount         []byte `json:"amount"`
	InitialBalance []byte `json:"initial-balance"`
	Ratio          string `json:"ratio"`
	Height         uint64 `json:"block-height"`
	BlockHash      string `json:"block-hash"`
	TotalTxs       uint64 `json:"total-txs"`
	TotalOps       uint64 `json:"total-ops"`
}

type CollectTxFee struct {
	Target    string `json:"target"`
	Amount    []byte `json:"amount"`
	Txs       uint64 `json:"txs"`
	Height    uint64 `json:"block-height"`
	BlockHash string `json:"block-hash"`
	TotalTxs  uint64 `json:"total-txs"`
	TotalOps  uint64 `json:"total-ops"`
}
