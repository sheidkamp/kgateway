package metrics

import (
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
		[]string{translatorNameLabel, "result"},
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
	translatorResourcesManaged = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Subsystem: translatorSubsystem,
			Name:      "resources_managed",
			Help:      "Current number of managed resources for translator",
		},
		[]string{translatorNameLabel, "namespace"},
	)
)

// TranslatorMetrics provides metrics for translator operations.
type TranslatorMetrics struct {
	translatorName      string
	translationsTotal   *prometheus.CounterVec
	translationDuration *prometheus.HistogramVec
	resourcesManaged    *prometheus.GaugeVec
}

// NewTranslatorMetrics creates a new TranslatorMetrics instance.
func NewTranslatorMetrics(translatorName string) *TranslatorMetrics {
	m := &TranslatorMetrics{
		translatorName:      translatorName,
		translationsTotal:   translationsTotal,
		translationDuration: translationDuration,
		resourcesManaged:    translatorResourcesManaged,
	}

	return m
}

// TranslationStart is called at the start of a translation function to begin metrics
// collection and returns a function called at the end to complete metrics recording.
func (m *TranslatorMetrics) TranslationStart() func(error) {
	start := time.Now()

	return func(err error) {
		duration := time.Since(start)

		m.translationDuration.WithLabelValues(m.translatorName).Observe(duration.Seconds())

		result := "success"
		if err != nil {
			result = "error"
		}

		m.translationsTotal.WithLabelValues(m.translatorName, result).Inc()
	}
}

// SetResourceCount updates the resource count gauge.
func (m *TranslatorMetrics) SetResourceCount(namespace string, count int) {
	m.resourcesManaged.WithLabelValues(m.translatorName, namespace).Set(float64(count))
}

// IncResourceCount increments the resource count gauge.
func (m *TranslatorMetrics) IncResourceCount(namespace string) {
	m.resourcesManaged.WithLabelValues(m.translatorName, namespace).Inc()
}

// DecResourceCount decrements the resource count gauge.
func (m *TranslatorMetrics) DecResourceCount(namespace string) {
	m.resourcesManaged.WithLabelValues(m.translatorName, namespace).Dec()
}

// ResetResourceCounts clears all resource counts.
func (m *TranslatorMetrics) ResetResourceCounts() {
	m.resourcesManaged.Reset()
}
