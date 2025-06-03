package metrics

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	collectionSubsystem = "collection"
	collectionNameLabel = "collection"
)

var (
	transformsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Subsystem: collectionSubsystem,
			Name:      "transforms_total",
			Help:      "Total transforms",
		},
		[]string{collectionNameLabel, "result"},
	)
	transformDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace:                       metricsNamespace,
			Subsystem:                       collectionSubsystem,
			Name:                            "transform_duration_seconds",
			Help:                            "Transform duration",
			NativeHistogramBucketFactor:     1.1,
			NativeHistogramMaxBucketNumber:  100,
			NativeHistogramMinResetDuration: time.Hour,
		},
		[]string{collectionNameLabel},
	)
	collectionResources = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Subsystem: collectionSubsystem,
			Name:      "resources",
			Help:      "Current number of resources managed by the collection",
		},
		[]string{collectionNameLabel, "name", "namespace"},
	)
)

// CollectionRecorder defines the interface for recording collection metrics.
type CollectionRecorder interface {
	TransformStart() func(error)
	ResetResources(namespace, name string)
	SetResources(namespace, name string, count int)
	IncResources(namespace, name string)
	DecResources(namespace, name string)
}

// collectionMetrics records metrics for collection operations.
type collectionMetrics struct {
	collectionName    string
	transformsTotal   *prometheus.CounterVec
	transformDuration *prometheus.HistogramVec
	resources         *prometheus.GaugeVec
	resourceNames     map[string]map[string]struct{}
	resourcesLock     sync.Mutex
}

// NewCollectionRecorder creates a new recorder for collection metrics.
func NewCollectionRecorder(collectionName string) CollectionRecorder {
	m := &collectionMetrics{
		collectionName:    collectionName,
		transformsTotal:   transformsTotal,
		transformDuration: transformDuration,
		resources:         collectionResources,
		resourceNames:     make(map[string]map[string]struct{}),
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
func (m *collectionMetrics) ResetResources(namespace, name string) {
	m.resourcesLock.Lock()
	defer m.resourcesLock.Unlock()

	if _, exists := m.resourceNames[namespace]; !exists {
		return
	}

	for name := range m.resourceNames[namespace] {
		m.resources.WithLabelValues(m.collectionName, name, namespace).Set(0)
	}

	delete(m.resourceNames[namespace], name)

	if len(m.resourceNames[namespace]) == 0 {
		delete(m.resourceNames, namespace)
	}
}

// SetResources updates the resource count gauge.
func (m *collectionMetrics) SetResources(namespace, name string, count int) {
	m.resourcesLock.Lock()
	defer m.resourcesLock.Unlock()

	if _, exists := m.resourceNames[namespace]; !exists {
		m.resourceNames[namespace] = make(map[string]struct{})
	}

	m.resourceNames[namespace][name] = struct{}{}

	m.resources.WithLabelValues(m.collectionName, name, namespace).Set(float64(count))
}

// IncResources increments the resource count gauge.
func (m *collectionMetrics) IncResources(namespace, name string) {
	m.resourcesLock.Lock()
	defer m.resourcesLock.Unlock()

	if _, exists := m.resourceNames[namespace]; !exists {
		m.resourceNames[namespace] = make(map[string]struct{})
	}

	m.resourceNames[namespace][name] = struct{}{}

	m.resources.WithLabelValues(m.collectionName, name, namespace).Inc()
}

// DecResources decrements the resource count gauge.
func (m *collectionMetrics) DecResources(namespace, name string) {
	m.resourcesLock.Lock()
	defer m.resourcesLock.Unlock()

	if _, exists := m.resourceNames[namespace]; !exists {
		m.resourceNames[namespace] = make(map[string]struct{})
	}

	m.resourceNames[namespace][name] = struct{}{}

	m.resources.WithLabelValues(m.collectionName, name, namespace).Dec()
}

// ResetCollectionMetrics resets the collection metrics.
func ResetCollectionMetrics() {
	transformsTotal.Reset()
	transformDuration.Reset()
	collectionResources.Reset()
}

// GetTransformsTotal returns the transforms counter.
func GetTransformsTotal() *prometheus.CounterVec {
	return transformsTotal
}

// GetTransformDuration returns the transform duration histogram.
func GetTransformDuration() *prometheus.HistogramVec {
	return transformDuration
}

// GetCollectionResources returns the collection resource count gauge.
func GetCollectionResources() *prometheus.GaugeVec {
	return collectionResources
}
