// Package metrics provides metrics for service operations.
package metrics

import (
	"errors"

	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

const metricsNamespace = "kgateway"

var (
	metricsToRegister = map[string]prometheus.Collector{
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
		"translatorResources":  translatorResources,
		"snapshotSyncsTotal":   snapshotSyncsTotal,
		"snapshotSyncDuration": snapshotSyncDuration,
		"snapshotResources":    snapshotResources,
		"routeListenerCounts":  routeListenerCounts,
		"routeDomainCounts":    routeDomainCounts,
	}
)

func registerMetrics() {
	for name, metric := range metricsToRegister {
		if err := metrics.Registry.Register(metric); err != nil &&
			!errors.Is(err, &prometheus.AlreadyRegisteredError{}) {
			panic("failed to register " + name + " metric: " + err.Error())
		}
	}
}

func init() {
	registerMetrics()
}
