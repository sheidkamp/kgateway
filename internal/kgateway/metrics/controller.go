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
	startupsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Subsystem: controllerSubsystem,
			Name:      "startups_total",
			Help:      "Total controller startups",
		},
		[]string{controllerNameLabel, "start_time"},
	)
)

// ControllerRecorder defines the interface for recording controller metrics.
type ControllerRecorder interface {
	ReconcileStart() func(error)
	IncStartups()
}

// controllerMetrics provides metrics for controller operations.
type controllerMetrics struct {
	controllerName       string
	reconciliationsTotal *prometheus.CounterVec
	reconcileDuration    *prometheus.HistogramVec
	startupsTotal        *prometheus.CounterVec
}

// NewControllerRecorder creates a new ControllerMetrics instance.
func NewControllerRecorder(controllerName string) ControllerRecorder {
	m := &controllerMetrics{
		controllerName:       controllerName,
		reconciliationsTotal: reconciliationsTotal,
		reconcileDuration:    reconcileDuration,
		startupsTotal:        startupsTotal,
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

// IncStartups increments the startup counter for the controller.
func (m *controllerMetrics) IncStartups() {
	startupTime := time.Now().Format(time.RFC3339)
	m.startupsTotal.WithLabelValues(m.controllerName, startupTime).Inc()
}

// ResetControllerMetrics resets the controller metrics.
func ResetControllerMetrics() {
	reconciliationsTotal.Reset()
	reconcileDuration.Reset()
	startupsTotal.Reset()
}

// GetReconciliationsTotal returns the reconciliations counter.
func GetReconciliationsTotal() *prometheus.CounterVec {
	return reconciliationsTotal
}

// GetReconcileDuration returns the reconcile duration histogram.
func GetReconcileDuration() *prometheus.HistogramVec {
	return reconcileDuration
}

// GetStartups total returns the controller startups counter.
func GetStartups() *prometheus.CounterVec {
	return startupsTotal
}
