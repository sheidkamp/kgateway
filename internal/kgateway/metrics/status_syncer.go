package metrics

import (
	"sync"
	"time"

	"github.com/kgateway-dev/kgateway/v2/pkg/metrics"
)

const (
	statusSubsystem = "status_syncer"
	syncerNameLabel = "syncer"
)

var (
	statusSyncsTotal = metrics.NewCounter(
		metrics.CounterOpts{
			Namespace: MetricsNamespace,
			Subsystem: statusSubsystem,
			Name:      "status_syncs_total",
			Help:      "Total status syncs",
		},
		[]string{syncerNameLabel, "result"},
	)
	statusSyncDuration = metrics.NewHistogram(
		metrics.HistogramOpts{
			Namespace:                       MetricsNamespace,
			Subsystem:                       statusSubsystem,
			Name:                            "status_sync_duration_seconds",
			Help:                            "Status sync duration",
			NativeHistogramBucketFactor:     1.1,
			NativeHistogramMaxBucketNumber:  100,
			NativeHistogramMinResetDuration: time.Hour,
		},
		[]string{syncerNameLabel},
	)
	statusSyncResources = metrics.NewGauge(
		metrics.GaugeOpts{
			Namespace: MetricsNamespace,
			Subsystem: statusSubsystem,
			Name:      "resources",
			Help:      "Current number of resources managed by the status syncer",
		},
		[]string{syncerNameLabel, "name", "namespace", "resource"},
	)
)

// StatusSyncResourcesLabels defines the labels for the syncer resources metric.
type StatusSyncResourcesLabels struct {
	Name      string
	Namespace string
	Resource  string
}

func (r StatusSyncResourcesLabels) toMetricsLabels(syncer string) []metrics.Label {
	return []metrics.Label{
		{Name: syncerNameLabel, Value: syncer},
		{Name: "name", Value: r.Name},
		{Name: "namespace", Value: r.Namespace},
		{Name: "resource", Value: r.Resource},
	}
}

// StatusSyncRecorder defines the interface for recording status syncer metrics.
type StatusSyncRecorder interface {
	StatusSyncStart() func(error)
	ResetResources(resource string)
	SetResources(labels StatusSyncResourcesLabels, count int)
	IncResources(labels StatusSyncResourcesLabels)
	DecResources(labels StatusSyncResourcesLabels)
}

// statusSyncMetrics records metrics for status syncer operations.
type statusSyncMetrics struct {
	syncerName         string
	statusSyncsTotal   metrics.Counter
	statusSyncDuration metrics.Histogram
	resources          metrics.Gauge
	resourceNames      map[string]map[string]map[string]struct{}
	resourcesLock      sync.Mutex
}

// NewStatusSyncRecorder creates a new recorder for status syncer metrics.
func NewStatusSyncRecorder(syncerName string) StatusSyncRecorder {
	m := &statusSyncMetrics{
		syncerName:         syncerName,
		statusSyncsTotal:   statusSyncsTotal,
		statusSyncDuration: statusSyncDuration,
		resources:          statusSyncResources,
		resourceNames:      make(map[string]map[string]map[string]struct{}),
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

		m.statusSyncDuration.Observe(duration.Seconds(),
			metrics.Label{Name: syncerNameLabel, Value: m.syncerName})

		result := "success"
		if err != nil {
			result = "error"
		}

		m.statusSyncsTotal.Inc([]metrics.Label{
			{Name: syncerNameLabel, Value: m.syncerName},
			{Name: "result", Value: result},
		}...)
	}
}

// ResetResources resets the resource count gauge for a specified resource.
func (m *statusSyncMetrics) ResetResources(resource string) {
	m.resourcesLock.Lock()

	namespaces, exists := m.resourceNames[resource]
	if !exists {
		m.resourcesLock.Unlock()

		return
	}

	delete(m.resourceNames, resource)

	m.resourcesLock.Unlock()

	for namespace, names := range namespaces {
		for name := range names {
			m.resources.Set(0, []metrics.Label{
				{Name: syncerNameLabel, Value: m.syncerName},
				{Name: "name", Value: name},
				{Name: "namespace", Value: namespace},
				{Name: "resource", Value: resource},
			}...)
		}
	}
}

// updateResourceNames updates the internal map of resource names.
func (m *statusSyncMetrics) updateResourceNames(labels StatusSyncResourcesLabels) {
	m.resourcesLock.Lock()

	if _, exists := m.resourceNames[labels.Resource]; !exists {
		m.resourceNames[labels.Resource] = make(map[string]map[string]struct{})
	}

	if _, exists := m.resourceNames[labels.Resource][labels.Namespace]; !exists {
		m.resourceNames[labels.Resource][labels.Namespace] = make(map[string]struct{})
	}

	m.resourceNames[labels.Resource][labels.Namespace][labels.Name] = struct{}{}

	m.resourcesLock.Unlock()
}

// SetResources updates the resource count gauge.
func (m *statusSyncMetrics) SetResources(labels StatusSyncResourcesLabels, count int) {
	m.updateResourceNames(labels)

	m.resources.Set(float64(count), labels.toMetricsLabels(m.syncerName)...)
}

// IncResources increments the resource count gauge.
func (m *statusSyncMetrics) IncResources(labels StatusSyncResourcesLabels) {
	m.updateResourceNames(labels)

	m.resources.Add(1, labels.toMetricsLabels(m.syncerName)...)
}

// DecResources decrements the resource count gauge.
func (m *statusSyncMetrics) DecResources(labels StatusSyncResourcesLabels) {
	m.updateResourceNames(labels)

	m.resources.Sub(1, labels.toMetricsLabels(m.syncerName)...)
}

// GetStatusSyncsTotal returns the status syncs counter.
// This is provided for testing purposes.
func GetStatusSyncsTotal() metrics.Counter {
	return statusSyncsTotal
}

// GetStatusSyncDuration returns the status sync duration histogram.
// This is provided for testing purposes.
func GetStatusSyncDuration() metrics.Histogram {
	return statusSyncDuration
}

// GetStatusSyncResources returns the status syncer resource count gauge.
// This is provided for testing purposes.
func GetStatusSyncResources() metrics.Gauge {
	return statusSyncResources
}
