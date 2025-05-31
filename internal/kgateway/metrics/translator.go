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
	translatorResourceCount = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Subsystem: translatorSubsystem,
			Name:      "resource_count",
			Help:      "Current number of resources managed by the translator",
		},
		[]string{"namespace", "resource", translatorNameLabel},
	)
)

// TranslatorRecorder defines the interface for recording translator metrics.
type TranslatorRecorder interface {
	TranslationStart() func(error)
	SetResourceCount(namespace, resource string, count int)
	IncResourceCount(namespace, resource string)
	DecResourceCount(namespace, resource string)
}

// translatorMetrics records metrics for translator operations.
type translatorMetrics struct {
	translatorName      string
	translationsTotal   *prometheus.CounterVec
	translationDuration *prometheus.HistogramVec
	resourceCount       *prometheus.GaugeVec
}

// NewTranslatorRecorder creates a new recorder for translator metrics.
func NewTranslatorRecorder(translatorName string) TranslatorRecorder {
	m := &translatorMetrics{
		translatorName:      translatorName,
		translationsTotal:   translationsTotal,
		translationDuration: translationDuration,
		resourceCount:       translatorResourceCount,
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

// SetResourceCount updates the resource count gauge.
func (m *translatorMetrics) SetResourceCount(namespace, resource string, count int) {
	m.resourceCount.WithLabelValues(namespace, resource, m.translatorName).Set(float64(count))
}

// IncResourceCount increments the resource count gauge.
func (m *translatorMetrics) IncResourceCount(namespace, resource string) {
	m.resourceCount.WithLabelValues(namespace, resource, m.translatorName).Inc()
}

// DecResourceCount decrements the resource count gauge.
func (m *translatorMetrics) DecResourceCount(namespace, resource string) {
	m.resourceCount.WithLabelValues(namespace, resource, m.translatorName).Dec()
}
