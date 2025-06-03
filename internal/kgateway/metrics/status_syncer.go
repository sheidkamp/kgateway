package metrics

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	statusSubsystem = "status_syncer"
	syncerNameLabel = "syncer"
)

var (
	statusSyncsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Subsystem: statusSubsystem,
			Name:      "status_syncs_total",
			Help:      "Total status syncs",
		},
		[]string{"result", syncerNameLabel},
	)
	statusSyncDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace:                       metricsNamespace,
			Subsystem:                       statusSubsystem,
			Name:                            "status_sync_duration_seconds",
			Help:                            "Status sync duration",
			NativeHistogramBucketFactor:     1.1,
			NativeHistogramMaxBucketNumber:  100,
			NativeHistogramMinResetDuration: time.Hour,
		},
		[]string{syncerNameLabel},
	)
	statusSyncResources = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Subsystem: statusSubsystem,
			Name:      "resources",
			Help:      "Current number of resources managed by the status syncer",
		},
		[]string{"namespace", "resource", syncerNameLabel},
	)
)

// StatusSyncRecorder defines the interface for recording status syncer metrics.
type StatusSyncRecorder interface {
	StatusSyncStart() func(error)
	ResetResources(resource string)
	SetResources(namespace, resource string, count int)
	IncResources(namespace, resource string)
	DecResources(namespace, resource string)
}

// statusSyncMetrics records metrics for status syncer operations.
type statusSyncMetrics struct {
	syncerName         string
	statusSyncsTotal   *prometheus.CounterVec
	statusSyncDuration *prometheus.HistogramVec
	resources          *prometheus.GaugeVec
	resourceNamespaces map[string]map[string]struct{}
	resourcesLock      sync.Mutex
}

// NewStatusSyncRecorder creates a new recorder for status syncer metrics.
func NewStatusSyncRecorder(syncerName string) StatusSyncRecorder {
	m := &statusSyncMetrics{
		syncerName:         syncerName,
		statusSyncsTotal:   statusSyncsTotal,
		statusSyncDuration: statusSyncDuration,
		resources:          statusSyncResources,
		resourceNamespaces: make(map[string]map[string]struct{}),
		resourcesLock:      sync.Mutex{},
	}

	return m
}

// StatusSyncStart is called at the start of a status sync function to begin metrics
// collection and returns a function called at the end to complete metrics recording.
func (m *statusSyncMetrics) StatusSyncStart() func(error) {
	start := time.Now()

	return func(err error) {
		duration := time.Since(start)

		m.statusSyncDuration.WithLabelValues(m.syncerName).Observe(duration.Seconds())

		result := "success"
		if err != nil {
			result = "error"
		}

		m.statusSyncsTotal.WithLabelValues(result, m.syncerName).Inc()
	}
}

// ResetResources resets the resource count gauge for a specified resource.
func (m *statusSyncMetrics) ResetResources(resource string) {
	m.resourcesLock.Lock()
	defer m.resourcesLock.Unlock()

	for namespace := range m.resourceNamespaces[resource] {
		m.resources.WithLabelValues(namespace, resource, m.syncerName).Set(0)
	}

	m.resourceNamespaces[resource] = make(map[string]struct{})
}

// SetResources updates the resource count gauge.
func (m *statusSyncMetrics) SetResources(namespace, resource string, count int) {
	m.resourcesLock.Lock()
	defer m.resourcesLock.Unlock()

	if _, exists := m.resourceNamespaces[resource]; !exists {
		m.resourceNamespaces[resource] = make(map[string]struct{})
	}

	m.resourceNamespaces[resource][namespace] = struct{}{}
	m.resources.WithLabelValues(namespace, resource, m.syncerName).Set(float64(count))
}

// IncResources increments the resource count gauge.
func (m *statusSyncMetrics) IncResources(namespace, resource string) {
	m.resources.WithLabelValues(namespace, resource, m.syncerName).Inc()
}

// DecResources decrements the resource count gauge.
func (m *statusSyncMetrics) DecResources(namespace, resource string) {
	m.resources.WithLabelValues(namespace, resource, m.syncerName).Dec()
}

// ResetStatusSyncMetrics resets the status syncer metrics.
func ResetStatusSyncMetrics() {
	statusSyncsTotal.Reset()
	statusSyncDuration.Reset()
	statusSyncResources.Reset()
}

// GetStatusSyncsTotal returns the status syncs counter.
func GetStatusSyncsTotal() *prometheus.CounterVec {
	return statusSyncsTotal
}

// GetStatusSyncDuration returns the status sync duration histogram.
func GetStatusSyncDuration() *prometheus.HistogramVec {
	return statusSyncDuration
}

// GetStatusSyncResources returns the status syncer resource count gauge.
func GetStatusSyncResources() *prometheus.GaugeVec {
	return statusSyncResources
}
