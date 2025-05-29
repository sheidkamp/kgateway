package metrics

import (
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
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
		translatorName: translatorName,
		translationTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "kgateway",
				Subsystem: "translator_" + translatorName,
				Name:      "translations_total",
				Help:      "Total translations for " + translatorName,
				ConstLabels: prometheus.Labels{
					"translator": translatorName,
				},
			},
			[]string{"result"},
		),
		translationDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "kgateway",
				Subsystem: "translator_" + translatorName,
				Name:      "translation_duration_seconds",
				Help:      "Translation duration for " + translatorName,
				ConstLabels: prometheus.Labels{
					"translator": translatorName,
				},
				NativeHistogramBucketFactor:     1.1,
				NativeHistogramMaxBucketNumber:  100,
				NativeHistogramMinResetDuration: time.Hour,
			},
			[]string{},
		),
		resourceTotal: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: "kgateway",
				Subsystem: "translator_" + translatorName,
				Name:      "resources_total",
				Help:      "Current number of managed resources for " + translatorName,
				ConstLabels: prometheus.Labels{
					"translator": translatorName,
				},
			},
			[]string{"namespace"},
		),
	}

	if err := metrics.Registry.Register(m.translationTotal); err != nil {
		fmt.Println("Failed to register translator metrics:", err)
	}

	if err := metrics.Registry.Register(m.translationDuration); err != nil {
		fmt.Println("Failed to register translator metrics:", err)
	}

	if err := metrics.Registry.Register(m.resourceTotal); err != nil {
		fmt.Println("Failed to register translator metrics:", err)
	}

	return m
}

// TranslationStart is called at the start of a translation function to begin metrics
// collection and returns a function called at the end to complete metrics recording.
func (m *TranslatorMetrics) TranslationStart() func(error) {
	start := time.Now()

	return func(err error) {
		duration := time.Since(start)

		m.translationDuration.WithLabelValues().Observe(duration.Seconds())

		result := "success"
		if err != nil {
			result = "error"
		}

		m.translationTotal.WithLabelValues(result).Inc()
	}
}

// SetResourceCount updates the resource count gauge.
func (m *TranslatorMetrics) SetResourceCount(namespace string, count int) {
	m.resourceTotal.WithLabelValues(namespace).Set(float64(count))
}

// IncResourceCount increments the resource count gauge.
func (m *TranslatorMetrics) IncResourceCount(namespace string) {
	m.resourceTotal.WithLabelValues(namespace).Inc()
}

// DecResourceCount decrements the resource count gauge.
func (m *TranslatorMetrics) DecResourceCount(namespace string) {
	m.resourceTotal.WithLabelValues(namespace).Dec()
}

// ResetResourceCounts clears all resource counts.
func (m *TranslatorMetrics) ResetResourceCounts() {
	m.resourceTotal.Reset()
}
