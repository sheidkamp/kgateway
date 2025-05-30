package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	reconciliationsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "kgateway",
			Subsystem: "controller",
			Name:      "reconciliations_total",
			Help:      "Total controller reconciliations",
		},
		[]string{"controller", "result"},
	)
	reconcileDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace:                       "kgateway",
			Subsystem:                       "controller",
			Name:                            "reconcile_duration_seconds",
			Help:                            "Reconcile duration for controller",
			NativeHistogramBucketFactor:     1.1,
			NativeHistogramMaxBucketNumber:  100,
			NativeHistogramMinResetDuration: time.Hour,
		},
		[]string{"controller"},
	)
	controllerResourcesManaged = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "kgateway",
			Subsystem: "controller",
			Name:      "resources_managed",
			Help:      "Current number of managed resources for controller",
		},
		[]string{"controller", "namespace"},
	)
)

// ControllerMetrics provides metrics for controller operations.
type ControllerMetrics struct {
	controllerName       string
	reconciliationsTotal *prometheus.CounterVec
	reconcileDuration    *prometheus.HistogramVec
	resourcesManaged     *prometheus.GaugeVec
}

// NewControllerMetrics creates a new ControllerMetrics instance.
func NewControllerMetrics(controllerName string) *ControllerMetrics {
	m := &ControllerMetrics{
		controllerName:       controllerName,
		reconciliationsTotal: reconciliationsTotal,
		reconcileDuration:    reconcileDuration,
		resourcesManaged:     controllerResourcesManaged,
	}

	return m
}

// ReconcileStart is called at the start of a controller reconciliation function
// to begin metrics collection and returns a function called at the end to
// complete metrics recording.
func (m *ControllerMetrics) ReconcileStart() func(error) {
	start := time.Now()

	return func(err error) {
		duration := time.Since(start)

		m.reconcileDuration.WithLabelValues(m.controllerName).Observe(duration.Seconds())

		result := "success"
		if err != nil {
			result = "error"
		}

		m.reconciliationsTotal.WithLabelValues(m.controllerName, result).Inc()
	}
}

// SetResourceCount updates the resource count gauge.
func (m *ControllerMetrics) SetResourceCount(namespace string, count int) {
	m.resourcesManaged.WithLabelValues(m.controllerName, namespace).Set(float64(count))
}

// IncResourceCount increments the resource count gauge.
func (m *ControllerMetrics) IncResourceCount(namespace string) {
	m.resourcesManaged.WithLabelValues(m.controllerName, namespace).Inc()
}

// DecResourceCount decrements the resource count gauge.
func (m *ControllerMetrics) DecResourceCount(namespace string) {
	m.resourcesManaged.WithLabelValues(m.controllerName, namespace).Dec()
}

// ResetResourceCounts clears all resource counts.
func (m *ControllerMetrics) ResetResourceCounts() {
	m.resourcesManaged.Reset()
}
