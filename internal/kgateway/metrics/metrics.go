// Package metrics provides metrics for service operations.
package metrics

import (
	"errors"

	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

const metricsNamespace = "kgateway"

func init() {
	// Register the metrics with the global registry.
	if err := metrics.Registry.Register(reconciliationsTotal); err != nil &&
		!errors.Is(err, &prometheus.AlreadyRegisteredError{}) {
		panic("failed to register reconciliationsTotal metric: " + err.Error())
	}

	if err := metrics.Registry.Register(reconcileDuration); err != nil &&
		!errors.Is(err, &prometheus.AlreadyRegisteredError{}) {
		panic("failed to register reconcileDuration metric: " + err.Error())
	}

	if err := metrics.Registry.Register(startupsTotal); err != nil &&
		!errors.Is(err, &prometheus.AlreadyRegisteredError{}) {
		panic("failed to register startupsTotal metric: " + err.Error())
	}

	if err := metrics.Registry.Register(translationsTotal); err != nil &&
		!errors.Is(err, &prometheus.AlreadyRegisteredError{}) {
		panic("failed to register translationsTotal metric: " + err.Error())
	}

	if err := metrics.Registry.Register(translationDuration); err != nil &&
		!errors.Is(err, &prometheus.AlreadyRegisteredError{}) {
		panic("failed to register translationDuration metric: " + err.Error())
	}
	if err := metrics.Registry.Register(translatorResources); err != nil &&
		!errors.Is(err, &prometheus.AlreadyRegisteredError{}) {
		panic("failed to register translatorResources metric: " + err.Error())
	}
	if err := metrics.Registry.Register(snapshotSyncsTotal); err != nil &&
		!errors.Is(err, &prometheus.AlreadyRegisteredError{}) {
		panic("failed to register snapshotSyncsTotal metric: " + err.Error())
	}
	if err := metrics.Registry.Register(snapshotSyncDuration); err != nil &&
		!errors.Is(err, &prometheus.AlreadyRegisteredError{}) {
		panic("failed to register snapshotSyncDuration metric: " + err.Error())
	}
	if err := metrics.Registry.Register(snapshotResources); err != nil &&
		!errors.Is(err, &prometheus.AlreadyRegisteredError{}) {
		panic("failed to register snapshotResources metric: " + err.Error())
	}
}
