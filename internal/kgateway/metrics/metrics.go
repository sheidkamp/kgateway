// Package metrics provides metrics for service operations.
package metrics

import (
	"errors"

	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

type metricsNamespace string

const (
	MetricsNamespaceKGateway metricsNamespace = "kgateway"
)

type metricsSubsystem string

const (
	MetricsSubsystemController    metricsSubsystem = "controller"
	MetricsSubsystemCollection    metricsSubsystem = "collection"
	MetricsSubsystemStatusSyncer  metricsSubsystem = "status-syncer"
	MetricsSubsystemTranslator    metricsSubsystem = "translator"
	MetricsSubsystemDomainMetrics metricsSubsystem = "domain-metrics"
	MetricsSubsystemRouting       metricsSubsystem = "routing"
)

// TODO: is this always the same as the subsystem?
type metricsLabel string

const (
	MetricsLabelController    metricsLabel = "controller"
	MetricsLabelCollection    metricsLabel = "collection"
	MetricsLabelStatusSyncer  metricsLabel = "status-syncer"
	MetricsLabelTranslator    metricsLabel = "translator"
	MetricsLabelDomainMetrics metricsLabel = "domain-metrics"
	MetricsLabelRouting       metricsLabel = "routing"
)

// MetricsRegistry stores registered metrics indexed by name.
type MetricsRegistry map[string]prometheus.Collector

var (
	metricsToRegister = MetricsRegistry{
		"transformsTotal":     transformsTotal,
		"transformDuration":   transformDuration,
		"collectionResources": collectionResources,
		// "reconciliationsTotal": reconciliationsTotal,
		// "reconcileDuration":    reconcileDuration,
		"statusSyncsTotal":    statusSyncsTotal,
		"statusSyncDuration":  statusSyncDuration,
		"statusSyncResources": statusSyncResources,
		"translationsTotal":   translationsTotal,
		"translationDuration": translationDuration,
		"translationsRunning": translationsRunning,
		//"domainsPerListener":  domainsPerListener,
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
