package metrics

import (
	"github.com/go-kit/kit/metrics"
	"github.com/go-kit/kit/metrics/discard"
	prometheus "github.com/go-kit/kit/metrics/prometheus"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
)

type TxPoolMetrics struct {
	Size metrics.Gauge
}

func (m *TxPoolMetrics) AddSize(delta int) {
	m.Size.Add(float64(delta))
}

func PromTxPoolMetrics() *TxPoolMetrics {
	return &TxPoolMetrics{
		Size: prometheus.NewGaugeFrom(stdprometheus.GaugeOpts{
			Namespace: Namespace,
			Subsystem: TxPoolSubsystem,
			Name:      "size",
			Help:      "Size of txpool.",
		}, []string{}),
	}
}

func NopTxPoolMetrics() *TxPoolMetrics {
	return &TxPoolMetrics{
		Size: discard.NewGauge(),
	}
}
