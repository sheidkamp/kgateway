// Package metrics provides metrics for service operations.
package metrics

import (
	"errors"

	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

type MetricsNamespace string

const (
	MetricsNamespaceKGateway MetricsNamespace = "kgateway"
)

type MetricsSubsystem string

const (
	MetricsSubsystemController    MetricsSubsystem = "controller"
	MetricsSubsystemCollection    MetricsSubsystem = "collection"
	MetricsSubsystemStatusSyncer  MetricsSubsystem = "status-syncer"
	MetricsSubsystemTranslator    MetricsSubsystem = "translator"
	MetricsSubsystemDomainMetrics MetricsSubsystem = "domain-metrics"
	MetricsSubsystemRouting       MetricsSubsystem = "routing"
)

type MetricsLabel string

const (
	MetricsLabelController    MetricsLabel = "controller"
	MetricsLabelCollection    MetricsLabel = "collection"
	MetricsLabelStatusSyncer  MetricsLabel = "status-syncer"
	MetricsLabelTranslator    MetricsLabel = "translator"
	MetricsLabelDomainMetrics MetricsLabel = "domain-metrics"
	MetricsLabelRouting       MetricsLabel = "routing"
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
