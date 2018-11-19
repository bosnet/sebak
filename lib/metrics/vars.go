package metrics

import (
	"github.com/go-kit/kit/metrics/discard"
)

var (
	Version   = discard.NewGauge()
	Consensus = NopConsensusMetrics()
	Sync      = NopSyncMetrics()
	TxPool    = NopTxPoolMetrics()
	API       = NopAPIMetrics()
)
