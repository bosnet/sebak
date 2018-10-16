package node

import (
	"encoding/json"
	"time"

	"boscoin.io/sebak/lib/common"
)

type NodeInfo struct {
	Node   NodeInfoNode  `json:"node"`
	Policy NodePolicy    `json:"policy"`
	Block  NodeBlockInfo `json:"block"`
}

type NodeInfoNode struct {
	Version    NodeVersion           `json:"version"`
	State      State                 `json:"state"`
	Alias      string                `json:"alias"`
	Address    string                `json:"address"`
	Endpoint   *common.Endpoint      `json:"endpoint"`
	Validators map[string]*Validator `json:"validators"`
}

type NodePolicy struct {
	NetworkID                 []byte        `json:"network-id"`
	InitialBalance            common.Amount `json:"initial-balance"`
	BaseReserve               common.Amount `json:"base-reserve"`
	BaseFee                   common.Amount `json:"base-fee"`
	BlockTime                 time.Duration `json:"block-time"`
	OperationsLimit           int           `json:"operations-limit"`
	TransactionsLimit         int           `json:"transactions-limit"`
	GenesisBlockConfirmedTime string        `json:"genesis-block-confirmed-time"`
	InflationRatio            string        `json:"inflation-ratio"`
	BlockHeightEndOfInflation uint64        `json:"block-height-end-of-inflation"`
}

type NodeBlockInfo struct {
	Height   uint64 `json:"height"`
	Hash     string `json:"hash"`
	TotalTxs uint64 `json:"total-txs"`
}

type NodeVersion struct {
	Version   string `json:"version"`
	GitCommit string `json:"git-commit"`
	GitState  string `json:"git-state"`
	BuildDate string `json:"build-date"`
}

func NewNodeInfoFromJSON(b []byte) (nodeInfo NodeInfo, err error) {
	err = json.Unmarshal(b, &nodeInfo)
	return
}
