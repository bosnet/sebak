package metrics

func InitPrometheusMetrics() {
	Version = PromVersion()
	Consensus = PromConsensusMetrics()
	Sync = PromSyncMetrics()
	TxPool = PromTxPoolMetrics()
	API = PromAPIMetrics()
}
