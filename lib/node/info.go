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
	NetworkID                 string        `json:"network-id"`                    // network id
	InitialBalance            common.Amount `json:"initial-balance"`               // initial balance of genesis account
	BaseReserve               common.Amount `json:"base-reserve"`                  // base reserve for one account
	BaseFee                   common.Amount `json:"base-fee"`                      // base fee of operation
	BlockTime                 time.Duration `json:"block-time"`                    // block creation time
	OperationsLimit           int           `json:"operations-limit"`              // operations limit in a transaction
	TransactionsLimit         int           `json:"transactions-limit"`            // transactions limit in a ballot
	GenesisBlockConfirmedTime string        `json:"genesis-block-confirmed-time"`  // confirmed time of genesis block; see `common.GenesisBlockConfirmedTime`
	InflationRatio            string        `json:"inflation-ratio"`               // inflation ratio; see `common.InflationRatio`
	UnfreezingPeriod          uint64        `json:"unfreezing-period"`             // unfreezing period
	BlockHeightEndOfInflation uint64        `json:"block-height-end-of-inflation"` // block height of inflation end; see `common.BlockHeightEndOfInflation`
}

type NodeBlockInfo struct {
	Height   uint64 `json:"height"`
	Hash     string `json:"hash"`
	TotalTxs uint64 `json:"total-txs"`
	TotalOps uint64 `json:"total-ops"`
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
