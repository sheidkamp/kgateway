package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	controllerSubsystem = "controller"
	controllerNameLabel = "controller"
)

var (
	reconciliationsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Subsystem: controllerSubsystem,
			Name:      "reconciliations_total",
			Help:      "Total controller reconciliations",
		},
		[]string{controllerNameLabel, "result"},
	)
	reconcileDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace:                       metricsNamespace,
			Subsystem:                       controllerSubsystem,
			Name:                            "reconcile_duration_seconds",
			Help:                            "Reconcile duration for controller",
			NativeHistogramBucketFactor:     1.1,
			NativeHistogramMaxBucketNumber:  100,
			NativeHistogramMinResetDuration: time.Hour,
		},
		[]string{controllerNameLabel},
	)
	controllerResourceCount = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Subsystem: controllerSubsystem,
			Name:      "resource_count",
			Help:      "Current number of resources managed by the controller",
		},
		[]string{controllerNameLabel, "namespace"},
	)
)

// ControllerRecorder defines the interface for recording controller metrics.
type ControllerRecorder interface {
	ReconcileStart() func(error)
	IncResourceCount(namespace string)
	DecResourceCount(namespace string)
	SetResourceCount(namespace string, count int)
	ResetResourceCounts()
}

// controllerMetrics provides metrics for controller operations.
type controllerMetrics struct {
	controllerName       string
	reconciliationsTotal *prometheus.CounterVec
	reconcileDuration    *prometheus.HistogramVec
	resourceCount        *prometheus.GaugeVec
}

// NewControllerRecorder creates a new ControllerMetrics instance.
func NewControllerRecorder(controllerName string) ControllerRecorder {
	m := &controllerMetrics{
		controllerName:       controllerName,
		reconciliationsTotal: reconciliationsTotal,
		reconcileDuration:    reconcileDuration,
		resourceCount:        controllerResourceCount,
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

		m.reconcileDuration.WithLabelValues(m.controllerName).Observe(duration.Seconds())

		result := "success"
		if err != nil {
			result = "error"
		}

		m.reconciliationsTotal.WithLabelValues(m.controllerName, result).Inc()
	}
}

// SetResourceCount updates the resource count gauge.
func (m *controllerMetrics) SetResourceCount(namespace string, count int) {
	m.resourceCount.WithLabelValues(m.controllerName, namespace).Set(float64(count))
}

// IncResourceCount increments the resource count gauge.
func (m *controllerMetrics) IncResourceCount(namespace string) {
	m.resourceCount.WithLabelValues(m.controllerName, namespace).Inc()
}

// DecResourceCount decrements the resource count gauge.
func (m *controllerMetrics) DecResourceCount(namespace string) {
	m.resourceCount.WithLabelValues(m.controllerName, namespace).Dec()
}

// ResetResourceCounts clears all resource counts.
func (m *controllerMetrics) ResetResourceCounts() {
	m.resourceCount.Reset()
}
