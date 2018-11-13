package metrics

import (
	"github.com/go-kit/kit/metrics"
	"github.com/go-kit/kit/metrics/discard"
	prometheus "github.com/go-kit/kit/metrics/prometheus"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
)

type ConsensusMetrics struct {
	Height metrics.Gauge
	Rounds metrics.Gauge
	NumTxs metrics.Gauge
}

func PromConsensusMetrics() *ConsensusMetrics {
	return &ConsensusMetrics{
		Height: prometheus.NewGaugeFrom(stdprometheus.GaugeOpts{
			Namespace: Namespace,
			Subsystem: ConsensusSubsystem,
			Name:      "height",
			Help:      "Height of the node.",
		}, []string{}),
		Rounds: prometheus.NewGaugeFrom(stdprometheus.GaugeOpts{
			Namespace: Namespace,
			Subsystem: ConsensusSubsystem,
			Name:      "rounds",
			Help:      "Number of rounds.",
		}, []string{}),
		NumTxs: prometheus.NewGaugeFrom(stdprometheus.GaugeOpts{
			Namespace: Namespace,
			Subsystem: ConsensusSubsystem,
			Name:      "num_txs",
			Help:      "Number of transactions.",
		}, []string{}),
	}
}

func NopConsensusMetrics() *ConsensusMetrics {
	return &ConsensusMetrics{
		Height: discard.NewGauge(),
		Rounds: discard.NewGauge(),
		NumTxs: discard.NewGauge(),
	}
}
