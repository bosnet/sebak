package metrics

import (
	"runtime"

	"boscoin.io/sebak/lib/version"

	"github.com/go-kit/kit/metrics"

	prometheus "github.com/go-kit/kit/metrics/prometheus"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
)

func PromVersion() metrics.Gauge {
	return prometheus.NewGaugeFrom(stdprometheus.GaugeOpts{
		Namespace: Namespace,
		Name:      "version",
		Help:      "Version of the node.",
	}, []string{"version", "git_commit", "go_version"})
}

func SetVersion() {
	Version.With(
		"version", version.Version,
		"git_commit", version.GitCommit,
		"go_version", runtime.Version()).Set(1)
}
