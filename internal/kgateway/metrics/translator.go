package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	translationsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "kgateway",
			Subsystem: "translator",
			Name:      "translations_total",
			Help:      "Total translations",
		},
		[]string{"translator", "result"},
	)
	translationDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace:                       "kgateway",
			Subsystem:                       "translator",
			Name:                            "translation_duration_seconds",
			Help:                            "Translation duration",
			NativeHistogramBucketFactor:     1.1,
			NativeHistogramMaxBucketNumber:  100,
			NativeHistogramMinResetDuration: time.Hour,
		},
		[]string{"translator"},
	)
	translatorResourcesTotal = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "kgateway",
			Subsystem: "translator",
			Name:      "resources_total",
			Help:      "Current number of managed resources for translator",
		},
		[]string{"translator", "namespace"},
	)
)

// TranslatorMetrics provides metrics for translator operations.
type TranslatorMetrics struct {
	translatorName      string
	translationTotal    *prometheus.CounterVec
	translationDuration *prometheus.HistogramVec
	resourceTotal       *prometheus.GaugeVec
}

// NewTranslatorMetrics creates a new TranslatorMetrics instance.
func NewTranslatorMetrics(translatorName string) *TranslatorMetrics {
	m := &TranslatorMetrics{
		translatorName:      translatorName,
		translationTotal:    translationsTotal,
		translationDuration: translationDuration,
		resourceTotal:       translatorResourcesTotal,
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

		m.translationTotal.WithLabelValues(m.translatorName, result).Inc()
	}
}

// SetResourceCount updates the resource count gauge.
func (m *TranslatorMetrics) SetResourceCount(namespace string, count int) {
	m.resourceTotal.WithLabelValues(m.translatorName, namespace).Set(float64(count))
}

// IncResourceCount increments the resource count gauge.
func (m *TranslatorMetrics) IncResourceCount(namespace string) {
	m.resourceTotal.WithLabelValues(m.translatorName, namespace).Inc()
}

// DecResourceCount decrements the resource count gauge.
func (m *TranslatorMetrics) DecResourceCount(namespace string) {
	m.resourceTotal.WithLabelValues(m.translatorName, namespace).Dec()
}

// ResetResourceCounts clears all resource counts.
func (m *TranslatorMetrics) ResetResourceCounts() {
	m.resourceTotal.Reset()
}
