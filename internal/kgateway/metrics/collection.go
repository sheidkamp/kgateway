package metrics

import (
	"sync"
	"time"

	"github.com/kgateway-dev/kgateway/v2/pkg/metrics"
)

const (
	collectionSubsystem = "collection"
	collectionNameLabel = "collection"
)

var (
	transformsTotal = metrics.NewCounter(
		metrics.CounterOpts{
			Namespace: MetricsNamespace,
			Subsystem: collectionSubsystem,
			Name:      "transforms_total",
			Help:      "Total transforms",
		},
		[]string{collectionNameLabel, "result"},
	)
	transformDuration = metrics.NewHistogram(
		metrics.HistogramOpts{
			Namespace:                       MetricsNamespace,
			Subsystem:                       collectionSubsystem,
			Name:                            "transform_duration_seconds",
			Help:                            "Transform duration",
			NativeHistogramBucketFactor:     1.1,
			NativeHistogramMaxBucketNumber:  100,
			NativeHistogramMinResetDuration: time.Hour,
		},
		[]string{collectionNameLabel},
	)
	collectionResources = metrics.NewGauge(
		metrics.GaugeOpts{
			Namespace: MetricsNamespace,
			Subsystem: collectionSubsystem,
			Name:      "resources",
			Help:      "Current number of resources managed by the collection",
		},
		[]string{collectionNameLabel, "name", "namespace", "resource"},
	)
)

// CollectionResourcesLabels defines the labels for the collection resources metric.
type CollectionResourcesLabels struct {
	Name      string
	Namespace string
	Resource  string
}

// toMetricsLabels converts CollectionResourcesLabels to a slice of metrics.Labels.
func (r CollectionResourcesLabels) toMetricsLabels(collection string) []metrics.Label {
	return []metrics.Label{
		{Name: collectionNameLabel, Value: collection},
		{Name: "name", Value: r.Name},
		{Name: "namespace", Value: r.Namespace},
		{Name: "resource", Value: r.Resource},
	}
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
	transformsTotal   metrics.Counter
	transformDuration metrics.Histogram
	resources         metrics.Gauge
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

		m.transformDuration.Observe(duration.Seconds(),
			metrics.Label{Name: collectionNameLabel, Value: m.collectionName})

		result := "success"
		if err != nil {
			result = "error"
		}

		m.transformsTotal.Inc([]metrics.Label{
			{Name: collectionNameLabel, Value: m.collectionName},
			{Name: "result", Value: result},
		}...)
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
			m.resources.Set(0, []metrics.Label{
				{Name: collectionNameLabel, Value: m.collectionName},
				{Name: "name", Value: name},
				{Name: "namespace", Value: namespace},
				{Name: "resource", Value: resource},
			}...)
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

	m.resources.Set(float64(count), labels.toMetricsLabels(m.collectionName)...)
}

// IncResources increments the resource count gauge.
func (m *collectionMetrics) IncResources(labels CollectionResourcesLabels) {
	m.updateResourceNames(labels)

	m.resources.Add(1, labels.toMetricsLabels(m.collectionName)...)
}

// DecResources decrements the resource count gauge.
func (m *collectionMetrics) DecResources(labels CollectionResourcesLabels) {
	m.updateResourceNames(labels)

	m.resources.Sub(1, labels.toMetricsLabels(m.collectionName)...)
}

// GetTransformsTotal returns the transforms counter.
// This is provided for testing purposes.
func GetTransformsTotal() metrics.Counter {
	return transformsTotal
}

// GetTransformDuration returns the transform duration histogram.
// This is provided for testing purposes.
func GetTransformDuration() metrics.Histogram {
	return transformDuration
}

// GetCollectionResources returns the collection resource count gauge.
// This is provided for testing purposes.
func GetCollectionResources() metrics.Gauge {
	return collectionResources
}
