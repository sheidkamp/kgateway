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
	translationsRunning = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Subsystem: translatorSubsystem,
			Name:      "translations_running",
			Help:      "Number of translations currently running",
		},
		[]string{translatorNameLabel},
	)
)

// TranslatorRecorder defines the interface for recording translator metrics.
type TranslatorRecorder interface {
	TranslationStart() func(error)
}

// translatorMetrics records metrics for translator operations.
type translatorMetrics struct {
	translatorName      string
	translationsTotal   *prometheus.CounterVec
	translationDuration *prometheus.HistogramVec
	translationsRunning *prometheus.GaugeVec
}

// NewTranslatorRecorder creates a new recorder for translator metrics.
func NewTranslatorRecorder(translatorName string) TranslatorRecorder {
	m := &translatorMetrics{
		translatorName:      translatorName,
		translationsTotal:   translationsTotal,
		translationDuration: translationDuration,
		translationsRunning: translationsRunning,
	}

	return m
}

// TranslationStart is called at the start of a translation function to begin metrics
// collection and returns a function called at the end to complete metrics recording.
func (m *translatorMetrics) TranslationStart() func(error) {
	start := time.Now()
	m.translationsRunning.WithLabelValues(m.translatorName).Inc()

	return func(err error) {
		duration := time.Since(start)

		m.translationDuration.WithLabelValues(m.translatorName).Observe(duration.Seconds())

		result := "success"
		if err != nil {
			result = "error"
		}

		m.translationsTotal.WithLabelValues(m.translatorName, result).Inc()
		m.translationsRunning.WithLabelValues(m.translatorName).Dec()
	}
}

// GetTranslationsTotal returns the translations counter.
// This is provided for testing purposes.
func GetTranslationsTotal() *prometheus.CounterVec {
	return translationsTotal
}

// GetTranslationDuration returns the translation duration histogram.
// This is provided for testing purposes.
func GetTranslationDuration() *prometheus.HistogramVec {
	return translationDuration
}

// GetTranslationsRunning returns the translations running gauge.
// This is provided for testing purposes.
func GetTranslationsRunning() *prometheus.GaugeVec {
	return translationsRunning
}
