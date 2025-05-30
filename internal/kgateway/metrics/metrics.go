// Package metrics provides metrics for service operations.
package metrics

import (
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

const metricsNamespace = "kgateway"

func init() {
	// Register the metrics with the global registry.
	metrics.Registry.MustRegister(
		reconciliationsTotal,
		reconcileDuration,
		controllerResourcesManaged,
		translationsTotal,
		translationDuration,
		translatorResourcesManaged,
	)
}
