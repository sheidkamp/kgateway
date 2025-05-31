package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	snapshotSubsystem = "snapshot"
	snapshotNameLabel = "snapshot"
)

var (
	snapshotSyncsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Subsystem: snapshotSubsystem,
			Name:      "syncs_total",
			Help:      "Total snapshot syncs",
		},
		[]string{"proxy", "result", snapshotNameLabel},
	)
	snapshotSyncDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace:                       metricsNamespace,
			Subsystem:                       snapshotSubsystem,
			Name:                            "sync_duration_seconds",
			Help:                            "Snapshot sync duration",
			NativeHistogramBucketFactor:     1.1,
			NativeHistogramMaxBucketNumber:  100,
			NativeHistogramMinResetDuration: time.Hour,
		},
		[]string{"proxy", snapshotNameLabel},
	)
	snapshotResourceCount = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Subsystem: snapshotSubsystem,
			Name:      "resource_count",
			Help:      "Current number of resources contained in the snapshot",
		},
		[]string{"proxy", snapshotNameLabel},
	)
)

// SnapshotRecorder defines the interface for recording snapshot metrics.
type SnapshotRecorder interface {
	SnapshotStart(proxy string) func(error)
	SetResourceCount(proxy string, count int)
	IncResourceCount(proxy string)
	DecResourceCount(proxy string)
}

// snapshotMetrics records metrics for snapshot operations.
type snapshotMetrics struct {
	snapshotName     string
	snapshotsTotal   *prometheus.CounterVec
	snapshotDuration *prometheus.HistogramVec
	resourceCount    *prometheus.GaugeVec
}

// NewSnapshotRecorder creates a new recorder for snapshot metrics.
func NewSnapshotRecorder(snapshotName string) SnapshotRecorder {
	m := &snapshotMetrics{
		snapshotName:     snapshotName,
		snapshotsTotal:   snapshotSyncsTotal,
		snapshotDuration: snapshotSyncDuration,
		resourceCount:    snapshotResourceCount,
	}

	return m
}

// SnapshotStart is called at the start of a snapshot function to begin metrics
// collection and returns a function called at the end to complete metrics recording.
func (m *snapshotMetrics) SnapshotStart(proxy string) func(error) {
	start := time.Now()

	return func(err error) {
		duration := time.Since(start)

		m.snapshotDuration.WithLabelValues(proxy, m.snapshotName).Observe(duration.Seconds())

		result := "success"
		if err != nil {
			result = "error"
		}

		m.snapshotsTotal.WithLabelValues(proxy, result, m.snapshotName).Inc()
	}
}

// SetResourceCount updates the resource count gauge.
func (m *snapshotMetrics) SetResourceCount(proxy string, count int) {
	m.resourceCount.WithLabelValues(proxy, m.snapshotName).Set(float64(count))
}

// IncResourceCount increments the resource count gauge.
func (m *snapshotMetrics) IncResourceCount(proxy string) {
	m.resourceCount.WithLabelValues(proxy, m.snapshotName).Inc()
}

// DecResourceCount decrements the resource count gauge.
func (m *snapshotMetrics) DecResourceCount(proxy string) {
	m.resourceCount.WithLabelValues(proxy, m.snapshotName).Dec()
}
