package metrics

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	translatorSubsystem = "translator"
	translatorNameLabel = "translator"
)

var (
	translationsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Subsystem: translatorSubsystem,
			Name:      "translations_total",
			Help:      "Total translations",
		},
		[]string{"result", translatorNameLabel},
	)
	translationDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace:                       metricsNamespace,
			Subsystem:                       translatorSubsystem,
			Name:                            "translation_duration_seconds",
			Help:                            "Translation duration",
			NativeHistogramBucketFactor:     1.1,
			NativeHistogramMaxBucketNumber:  100,
			NativeHistogramMinResetDuration: time.Hour,
		},
		[]string{translatorNameLabel},
	)
	translatorResources = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Subsystem: translatorSubsystem,
			Name:      "resources",
			Help:      "Current number of resources managed by the translator",
		},
		[]string{"name", "namespace", "resource", translatorNameLabel},
	)
)

// TranslatorRecorder defines the interface for recording translator metrics.
type TranslatorRecorder interface {
	TranslationStart() func(error)
	ResetResources(resource string)
	SetResources(namespace, name, resource string, count int)
	IncResources(namespace, name, resource string)
	DecResources(namespace, name, resource string)
}

// translatorMetrics records metrics for translator operations.
type translatorMetrics struct {
	translatorName      string
	translationsTotal   *prometheus.CounterVec
	translationDuration *prometheus.HistogramVec
	resources           *prometheus.GaugeVec
	resourceNames       map[string]map[string]map[string]struct{}
	resourcesLock       sync.Mutex
}

// NewTranslatorRecorder creates a new recorder for translator metrics.
func NewTranslatorRecorder(translatorName string) TranslatorRecorder {
	m := &translatorMetrics{
		translatorName:      translatorName,
		translationsTotal:   translationsTotal,
		translationDuration: translationDuration,
		resources:           translatorResources,
		resourceNames:       make(map[string]map[string]map[string]struct{}),
		resourcesLock:       sync.Mutex{},
	}

	return m
}

// TranslationStart is called at the start of a translation function to begin metrics
// collection and returns a function called at the end to complete metrics recording.
func (m *translatorMetrics) TranslationStart() func(error) {
	start := time.Now()

	return func(err error) {
		duration := time.Since(start)

		m.translationDuration.WithLabelValues(m.translatorName).Observe(duration.Seconds())

		result := "success"
		if err != nil {
			result = "error"
		}

		m.translationsTotal.WithLabelValues(result, m.translatorName).Inc()
	}
}

// ResetResources resets the resource count gauge for a specified resource.
func (m *translatorMetrics) ResetResources(resource string) {
	m.resourcesLock.Lock()
	defer m.resourcesLock.Unlock()

	for namespace, resources := range m.resourceNames {
		for name, resourceMap := range resources {
			if _, exists := resourceMap[resource]; !exists {
				continue
			}

			m.resources.WithLabelValues(name, namespace, resource, m.translatorName).Set(0)

			delete(m.resourceNames[namespace][name], resource)
			if len(m.resourceNames[namespace][name]) == 0 {
				delete(m.resourceNames[namespace], name)
			}
		}

		if len(m.resourceNames[namespace]) == 0 {
			delete(m.resourceNames, namespace)
		}
	}
}

// SetResources updates the resource count gauge.
func (m *translatorMetrics) SetResources(namespace, name, resource string, count int) {
	m.resourcesLock.Lock()
	defer m.resourcesLock.Unlock()

	if _, exists := m.resourceNames[namespace]; !exists {
		m.resourceNames[namespace] = make(map[string]map[string]struct{})
	}

	if _, exists := m.resourceNames[namespace][name]; !exists {
		m.resourceNames[namespace][name] = make(map[string]struct{})
	}

	m.resourceNames[namespace][name][resource] = struct{}{}

	m.resources.WithLabelValues(name, namespace, resource, m.translatorName).Set(float64(count))
}

// IncResources increments the resource count gauge.
func (m *translatorMetrics) IncResources(namespace, name, resource string) {
	m.resourcesLock.Lock()
	defer m.resourcesLock.Unlock()

	if _, exists := m.resourceNames[namespace]; !exists {
		m.resourceNames[namespace] = make(map[string]map[string]struct{})
	}

	if _, exists := m.resourceNames[namespace][name]; !exists {
		m.resourceNames[namespace][name] = make(map[string]struct{})
	}

	m.resourceNames[namespace][name][resource] = struct{}{}

	m.resources.WithLabelValues(name, namespace, resource, m.translatorName).Inc()
}

// DecResources decrements the resource count gauge.
func (m *translatorMetrics) DecResources(namespace, name, resource string) {
	m.resourcesLock.Lock()
	defer m.resourcesLock.Unlock()

	if _, exists := m.resourceNames[namespace]; !exists {
		m.resourceNames[namespace] = make(map[string]map[string]struct{})
	}

	if _, exists := m.resourceNames[namespace][name]; !exists {
		m.resourceNames[namespace][name] = make(map[string]struct{})
	}

	m.resourceNames[namespace][name][resource] = struct{}{}

	m.resources.WithLabelValues(name, namespace, resource, m.translatorName).Dec()
}

// ResetTranslatorMetrics resets the translator metrics.
func ResetTranslatorMetrics() {
	translationsTotal.Reset()
	translationDuration.Reset()
	translatorResources.Reset()
}

// GetTranslationsTotal returns the translations counter.
func GetTranslationsTotal() *prometheus.CounterVec {
	return translationsTotal
}

// GetTranslationDuration returns the translation duration histogram.
func GetTranslationDuration() *prometheus.HistogramVec {
	return translationDuration
}

// GetTranslatorResources returns the translator resource count gauge.
func GetTranslatorResources() *prometheus.GaugeVec {
	return translatorResources
}
