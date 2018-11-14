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

	TotalTxs metrics.Gauge
	TotalOps metrics.Gauge

	Validators        metrics.Gauge
	MissingValidators metrics.Gauge
}

func (c *ConsensusMetrics) SetHeight(height uint64) {
	c.Height.Set(float64(height))
}
func (c *ConsensusMetrics) SetRounds(round uint64) {
	c.Rounds.Set(float64(round))
}
func (c *ConsensusMetrics) SetTotalTxs(total uint64) {
	c.TotalTxs.Set(float64(total))
}
func (c *ConsensusMetrics) SetTotalOps(total uint64) {
	c.TotalOps.Set(float64(total))
}
func (c *ConsensusMetrics) SetValidators(num int) {
	c.Validators.Set(float64(num))
}
func (c *ConsensusMetrics) SetMissingValidators(num int) {
	c.MissingValidators.Set(float64(num))
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
		TotalTxs: prometheus.NewGaugeFrom(stdprometheus.GaugeOpts{
			Namespace: Namespace,
			Subsystem: ConsensusSubsystem,
			Name:      "total_txs",
			Help:      "Total number of transactions.",
		}, []string{}),
		TotalOps: prometheus.NewGaugeFrom(stdprometheus.GaugeOpts{
			Namespace: Namespace,
			Subsystem: ConsensusSubsystem,
			Name:      "total_ops",
			Help:      "Total number of operations.",
		}, []string{}),
		Validators: prometheus.NewGaugeFrom(stdprometheus.GaugeOpts{
			Namespace: Namespace,
			Subsystem: ConsensusSubsystem,
			Name:      "validators",
			Help:      "Number of validators",
		}, []string{}),
		MissingValidators: prometheus.NewGaugeFrom(stdprometheus.GaugeOpts{
			Namespace: Namespace,
			Subsystem: ConsensusSubsystem,
			Name:      "missing_validators",
			Help:      "Number of missing validators.",
		}, []string{}),
	}
}

func NopConsensusMetrics() *ConsensusMetrics {
	return &ConsensusMetrics{
		Height: discard.NewGauge(),
		Rounds: discard.NewGauge(),

		TotalTxs: discard.NewGauge(),
		TotalOps: discard.NewGauge(),

		Validators:        discard.NewGauge(),
		MissingValidators: discard.NewGauge(),
	}
}
