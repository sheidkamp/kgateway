package controller

import (
	"time"

	"github.com/kgateway-dev/kgateway/v2/internal/kgateway/metrics"
)

var (
	m                    = metrics.NewMetrics()
	reconciliationsTotal = m.NewCounterVec(metrics.CounterOpts{
		Namespace: metrics.MetricsNamespaceKGateway,
		Subsystem: metrics.MetricsSubsystemController,
		Name:      "reconciliations_total",
		Help:      "Total controller reconciliations",
	}, []string{string(metrics.MetricsLabelController), "result"})

	reconcileDuration = m.NewHistogramVec(
		metrics.HistogramOpts{
			Namespace:                       metrics.MetricsNamespaceKGateway,
			Subsystem:                       metrics.MetricsSubsystemController,
			Name:                            "reconcile_duration_seconds",
			Help:                            "Reconcile duration for controller",
			NativeHistogramBucketFactor:     1.1,
			NativeHistogramMaxBucketNumber:  100,
			NativeHistogramMinResetDuration: time.Hour,
		},
		[]string{string(metrics.MetricsLabelController)},
	)
)

func init() {
	m.RegisterMetrics([]metrics.Collector{
		metrics.Collector(reconciliationsTotal),
		metrics.Collector(reconcileDuration),
	})
}

// ControllerRecorder provides metrics for controller operations.
type ControllerRecorder struct {
	controllerName       string
	reconciliationsTotal metrics.CounterVec
	reconcileDuration    metrics.HistogramVec
	initialized          bool
}

// NewControllerRecorder creates a new ControllerMetrics instance.
func NewControllerRecorder(controllerName string) *ControllerRecorder {
	m := &ControllerRecorder{
		controllerName:       controllerName,
		reconciliationsTotal: reconciliationsTotal,
		reconcileDuration:    reconcileDuration,
		initialized:          true,
	}

	return m
}

// ReconcileStart is called at the start of a controller reconciliation function
// to begin metrics collection and returns a function called at the end to
// complete metrics recording.
func (m *ControllerRecorder) ReconcileStart() func(error) {
	if !m.initialized {
		return func(err error) {}
	}

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

// GetReconciliationsTotal returns the reconciliations counter.
// This is provided for testing purposes.
func GetReconciliationsTotal() metrics.CounterVec {
	return reconciliationsTotal
}

// GetReconcileDuration returns the reconcile duration histogram.
// This is provided for testing purposes.
func GetReconcileDuration() metrics.HistogramVec {
	return reconcileDuration
}
