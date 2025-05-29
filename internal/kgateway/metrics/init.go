// Package metrics provides metrics for service operations.
package metrics

import (
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

func init() {
	// Register the metrics with the global registry.
	metrics.Registry.MustRegister(
		reconciliationsTotal,
		reconcileDuration,
		controllerResourcesTotal,
		translationsTotal,
		translationDuration,
		translatorResourcesTotal,
	)
}
