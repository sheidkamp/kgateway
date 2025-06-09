package controller

import (
	"time"

	"github.com/kgateway-dev/kgateway/v2/pkg/metrics"
)

const (
	controllerSubsystem = "controller"
	controllerNameLabel = "controller"
)

var (
	reconciliationsTotal = metrics.NewCounter(
		metrics.CounterOpts{
			Subsystem: controllerSubsystem,
			Name:      "reconciliations_total",
			Help:      "Total controller reconciliations",
		},
		[]string{controllerNameLabel, "result"},
	)
	reconcileDuration = metrics.NewHistogram(
		metrics.HistogramOpts{
			Subsystem:                       controllerSubsystem,
			Name:                            "reconcile_duration_seconds",
			Help:                            "Reconcile duration for controller",
			NativeHistogramBucketFactor:     1.1,
			NativeHistogramMaxBucketNumber:  100,
			NativeHistogramMinResetDuration: time.Hour,
		},
		[]string{controllerNameLabel},
	)
)

// controllerMetricsRecorder defines the interface for recording controller metrics.
type controllerMetricsRecorder interface {
	ReconcileStart() func(error)
}

// controllerMetrics provides metrics for controller operations.
type controllerMetrics struct {
	controllerName       string
	reconciliationsTotal metrics.Counter
	reconcileDuration    metrics.Histogram
}

// NewControllerMetricsRecorder creates a new ControllerMetrics instance.
func NewControllerMetricsRecorder(controllerName string) controllerMetricsRecorder {
	m := &controllerMetrics{
		controllerName:       controllerName,
		reconciliationsTotal: reconciliationsTotal,
		reconcileDuration:    reconcileDuration,
	}

	return m
}

// ReconcileStart is called at the start of a controller reconciliation function
// to begin metrics collection and returns a function called at the end to
// complete metrics recording.
func (m *controllerMetrics) ReconcileStart() func(error) {
	start := time.Now()

	return func(err error) {
		duration := time.Since(start)

		m.reconcileDuration.Observe(duration.Seconds(),
			metrics.Label{Name: controllerNameLabel, Value: m.controllerName})

		result := "success"
		if err != nil {
			result = "error"
		}

		m.reconciliationsTotal.Inc([]metrics.Label{
			{Name: controllerNameLabel, Value: m.controllerName},
			{Name: "result", Value: result},
		}...)
	}
}

// GetReconciliationsTotalMetric returns the reconciliations counter.
// This is provided for testing purposes.
func GetReconciliationsTotalMetric() metrics.Counter {
	return reconciliationsTotal
}

// GetReconcileDurationMetric returns the reconcile duration histogram.
// This is provided for testing purposes.
func GetReconcileDurationMetric() metrics.Histogram {
	return reconcileDuration
}
