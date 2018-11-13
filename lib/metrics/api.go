package metrics

import (
	"github.com/go-kit/kit/metrics"
	"github.com/go-kit/kit/metrics/discard"
	prometheus "github.com/go-kit/kit/metrics/prometheus"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
)

type APIMetrics struct {
	RequestsTotal          metrics.Counter
	RequestErrorsTotal     metrics.Counter
	RequestDurationSeconds metrics.Histogram
}

func PromAPIMetrics() *APIMetrics {
	return &APIMetrics{
		RequestsTotal: prometheus.NewCounterFrom(stdprometheus.CounterOpts{
			Namespace: Namespace,
			Subsystem: APISubsystem,
			Name:      "requests_total",
			Help:      "Total number of requests.",
		}, []string{"endpoint", "method", "status"}),
		RequestErrorsTotal: prometheus.NewCounterFrom(stdprometheus.CounterOpts{
			Namespace: Namespace,
			Subsystem: APISubsystem,
			Name:      "request_errors_total",
			Help:      "Total number of request errors",
		}, []string{"endpoint", "method", "status"}),
		RequestDurationSeconds: prometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
			Namespace: Namespace,
			Subsystem: APISubsystem,
			Name:      "request_duration_seconds",
		}, []string{"endpoint", "method", "status"}),
	}
}

func NopAPIMetrics() *APIMetrics {
	return &APIMetrics{
		RequestsTotal:          discard.NewCounter(),
		RequestErrorsTotal:     discard.NewCounter(),
		RequestDurationSeconds: discard.NewHistogram(),
	}
}
