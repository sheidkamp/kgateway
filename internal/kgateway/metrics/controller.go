// Package metrics provides metrics for controller operations.
package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

// ControllerMetrics provides metrics for controller operations.
type ControllerMetrics struct {
	controllerName    string
	reconcileTotal    *prometheus.CounterVec
	reconcileDuration *prometheus.HistogramVec
	resourceTotal     *prometheus.GaugeVec
}

// NewControllerMetrics creates a new ControllerMetrics instance.
func NewControllerMetrics(controllerName string) *ControllerMetrics {
	m := &ControllerMetrics{
		controllerName: controllerName,
		reconcileTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "kgateway",
				Subsystem: "controller_" + controllerName,
				Name:      "reconciliations_total",
				Help:      "Total reconciliations for " + controllerName,
				ConstLabels: prometheus.Labels{
					"controller": controllerName,
				},
			},
			[]string{"result"},
		),
		reconcileDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "kgateway",
				Subsystem: "controller_" + controllerName,
				Name:      "reconcile_duration_seconds",
				Help:      "Reconcile duration for " + controllerName,
				ConstLabels: prometheus.Labels{
					"controller": controllerName,
				},
				NativeHistogramBucketFactor:     1.1,
				NativeHistogramMaxBucketNumber:  100,
				NativeHistogramMinResetDuration: time.Hour,
			},
			[]string{},
		),
		resourceTotal: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: "kgateway",
				Subsystem: "controller_" + controllerName,
				Name:      "resources_total",
				Help:      "Current number of managed resources for " + controllerName,
				ConstLabels: prometheus.Labels{
					"controller": controllerName,
				},
			},
			[]string{"namespace"},
		),
	}

	metrics.Registry.MustRegister(
		m.reconcileTotal,
		m.reconcileDuration,
		m.resourceTotal,
	)

	return m
}

// ReconcileStart is called at the start of a controller reconciliation function
// to begin metrics collection and returns a function called at the end to
// complete metrics recording.
func (m *ControllerMetrics) ReconcileStart() func(error) {
	start := time.Now()

	return func(err error) {
		duration := time.Since(start)

		m.reconcileDuration.WithLabelValues().Observe(duration.Seconds())

		result := "success"
		if err != nil {
			result = "error"
		}

		m.reconcileTotal.WithLabelValues(result).Inc()
	}
}

// SetResourceCount updates the resource count gauge.
func (m *ControllerMetrics) SetResourceCount(namespace string, count int) {
	m.resourceTotal.WithLabelValues(namespace).Set(float64(count))
}

// IncResourceCount increments the resource count gauge.
func (m *ControllerMetrics) IncResourceCount(namespace string) {
	m.resourceTotal.WithLabelValues(namespace).Inc()
}

// DecResourceCount decrements the resource count gauge.
func (m *ControllerMetrics) DecResourceCount(namespace string) {
	m.resourceTotal.WithLabelValues(namespace).Dec()
}

// ResetResourceCounts clears all resource counts.
func (m *ControllerMetrics) ResetResourceCounts() {
	m.resourceTotal.Reset()
}
