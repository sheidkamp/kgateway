package metrics

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	transformsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: string(MetricsNamespaceKGateway),
			Subsystem: string(MetricsSubsystemCollection),
			Name:      "transforms_total",
			Help:      "Total transforms",
		},
		[]string{string(MetricsLabelCollection), "result"},
	)
	transformDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace:                       string(MetricsNamespaceKGateway),
			Subsystem:                       string(MetricsSubsystemCollection),
			Name:                            "transform_duration_seconds",
			Help:                            "Transform duration",
			NativeHistogramBucketFactor:     1.1,
			NativeHistogramMaxBucketNumber:  100,
			NativeHistogramMinResetDuration: time.Hour,
		},
		[]string{string(MetricsLabelCollection)},
	)
	collectionResources = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: string(MetricsNamespaceKGateway),
			Subsystem: string(MetricsSubsystemCollection),
			Name:      "resources",
			Help:      "Current number of resources managed by the collection",
		},
		[]string{string(MetricsLabelCollection), "name", "namespace", "resource"},
	)
)

// CollectionResourcesLabels defines the labels for the collection resources metric.
type CollectionResourcesLabels struct {
	Name      string
	Namespace string
	Resource  string
}

func (r CollectionResourcesLabels) toMetricsLabels(collection string) []string {
	return []string{collection, r.Name, r.Namespace, r.Resource}
}

// CollectionRecorder defines the interface for recording collection metrics.
type CollectionRecorder interface {
	TransformStart() func(error)
	ResetResources(resource string)
	SetResources(labels CollectionResourcesLabels, count int)
	IncResources(labels CollectionResourcesLabels)
	DecResources(labels CollectionResourcesLabels)
}

// collectionMetrics records metrics for collection operations.
type collectionMetrics struct {
	collectionName    string
	transformsTotal   *prometheus.CounterVec
	transformDuration *prometheus.HistogramVec
	resources         *prometheus.GaugeVec
	resourceNames     map[string]map[string]map[string]struct{}
	resourcesLock     sync.Mutex
}

// NewCollectionRecorder creates a new recorder for collection metrics.
func NewCollectionRecorder(collectionName string) CollectionRecorder {
	m := &collectionMetrics{
		collectionName:    collectionName,
		transformsTotal:   transformsTotal,
		transformDuration: transformDuration,
		resources:         collectionResources,
		resourceNames:     make(map[string]map[string]map[string]struct{}),
		resourcesLock:     sync.Mutex{},
	}

	return m
}

// TransformStart is called at the start of a transform function to begin metrics
// collection and returns a function called at the end to complete metrics recording.
func (m *collectionMetrics) TransformStart() func(error) {
	start := time.Now()

	return func(err error) {
		duration := time.Since(start)

		m.transformDuration.WithLabelValues(m.collectionName).Observe(duration.Seconds())

		result := "success"
		if err != nil {
			result = "error"
		}

		m.transformsTotal.WithLabelValues(m.collectionName, result).Inc()
	}
}

// ResetResources resets the resource count gauge for a specified resource.
func (m *collectionMetrics) ResetResources(resource string) {
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
			m.resources.WithLabelValues(m.collectionName, name, namespace, resource).Set(0)
		}
	}
}

// updateResourceNames updates the internal map of resource names.
func (m *collectionMetrics) updateResourceNames(labels CollectionResourcesLabels) {
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
func (m *collectionMetrics) SetResources(labels CollectionResourcesLabels, count int) {
	m.updateResourceNames(labels)

	m.resources.WithLabelValues(labels.toMetricsLabels(m.collectionName)...).Set(float64(count))
}

// IncResources increments the resource count gauge.
func (m *collectionMetrics) IncResources(labels CollectionResourcesLabels) {
	m.updateResourceNames(labels)

	m.resources.WithLabelValues(labels.toMetricsLabels(m.collectionName)...).Inc()
}

// DecResources decrements the resource count gauge.
func (m *collectionMetrics) DecResources(labels CollectionResourcesLabels) {
	m.updateResourceNames(labels)

	m.resources.WithLabelValues(labels.toMetricsLabels(m.collectionName)...).Dec()
}

// GetTransformsTotal returns the transforms counter.
// This is provided for testing purposes.
func GetTransformsTotal() *prometheus.CounterVec {
	return transformsTotal
}

// GetTransformDuration returns the transform duration histogram.
// This is provided for testing purposes.
func GetTransformDuration() *prometheus.HistogramVec {
	return transformDuration
}

// GetCollectionResources returns the collection resource count gauge.
// This is provided for testing purposes.
func GetCollectionResources() *prometheus.GaugeVec {
	return collectionResources
}
