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
	SequenceID int    `json:"sequence_id"`
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
		Accounts   Link `json:"accounts"`
		Operations Link `json:"operations"`
	} `json:"_links"`
	Hash           string `json:"hash"`
	Source         string `json:"source"`
	Fee            string `json:"fee"`
	SequenceID     string `json:"sequence_id"`
	Created        string `json:"created"`
	OperationCount uint64 `json:"operation_count"`
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
		Self         Link `json:"self"`
		Transactions Link `json:"transactions"`
	} `json:"_links"`
	Hash   string `json:"hash"`
	Source string `json:"source"`
	Type   string `json:"type"`
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
