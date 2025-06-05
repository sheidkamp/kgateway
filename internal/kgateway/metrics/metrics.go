// Package metrics provides metrics for service operations.
package metrics

import (
	"errors"

	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

const metricsNamespace = "kgateway"

// MetricsRegistry stores registered metrics indexed by name.
type MetricsRegistry map[string]prometheus.Collector

var (
	metricsToRegister = MetricsRegistry{
		"transformsTotal":      transformsTotal,
		"transformDuration":    transformDuration,
		"collectionResources":  collectionResources,
		"reconciliationsTotal": reconciliationsTotal,
		"reconcileDuration":    reconcileDuration,
		"startupsTotal":        startupsTotal,
		"statusSyncsTotal":     statusSyncsTotal,
		"statusSyncDuration":   statusSyncDuration,
		"statusSyncResources":  statusSyncResources,
		"translationsTotal":    translationsTotal,
		"translationDuration":  translationDuration,
	}
)

func init() {
	RegisterMetrics(metricsToRegister)
}

// RegisterMetrics registers a map of metrics with the controller-runtime metrics registry.
func RegisterMetrics(registry MetricsRegistry) {
	for name, metric := range registry {
		if err := metrics.Registry.Register(metric); err != nil &&
			!errors.Is(err, &prometheus.AlreadyRegisteredError{}) {
			panic("failed to register " + name + " metric: " + err.Error())
		}
	}
}
