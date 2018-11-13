package metrics

import (
	"github.com/go-kit/kit/metrics"
	"github.com/go-kit/kit/metrics/discard"
	prometheus "github.com/go-kit/kit/metrics/prometheus"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
)

type SyncMetrics struct {
	Height          metrics.Gauge
	ErrorTotal      metrics.Counter
	DurationSeconds metrics.Histogram
}

func (s *SyncMetrics) SetHeight(height uint64) {
	s.Height.Set(float64(height))
}

func (s *SyncMetrics) AddFetchError() {
	s.ErrorTotal.With("component", "fetcher").Add(1)
}
func (s *SyncMetrics) AddValidateError() {
	s.ErrorTotal.With("component", "validator").Add(1)
}

func PromSyncMetrics() *SyncMetrics {
	return &SyncMetrics{
		Height: prometheus.NewGaugeFrom(stdprometheus.GaugeOpts{
			Namespace: Namespace,
			Subsystem: SyncSubsystem,
			Name:      "height",
			Help:      "Height of sync.",
		}, []string{}),
		ErrorTotal: prometheus.NewCounterFrom(stdprometheus.CounterOpts{
			Namespace: Namespace,
			Subsystem: SyncSubsystem,
			Name:      "error_total",
			Help:      "Number of failed sync.",
		}, []string{"component"}),
		DurationSeconds: prometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
			Namespace: Namespace,
			Subsystem: SyncSubsystem,
			Name:      "duration_seconds",
			Help:      "Time processing one block.",
		}, []string{}),
	}
}

func NopSyncMetrics() *SyncMetrics {
	return &SyncMetrics{
		Height:          discard.NewGauge(),
		ErrorTotal:      discard.NewCounter(),
		DurationSeconds: discard.NewHistogram(),
	}
}
