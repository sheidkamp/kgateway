package metrics

import (
	"time"

	"github.com/kgateway-dev/kgateway/v2/pkg/metrics"
)

const (
	translatorSubsystem = "translator"
	translatorNameLabel = "translator"
	routingSubsystem    = "routing"
)

var (
	translationsTotal = metrics.NewCounter(
		metrics.CounterOpts{
			Subsystem: translatorSubsystem,
			Name:      "translations_total",
			Help:      "Total translations",
		},
		[]string{translatorNameLabel, "result"},
	)
	translationDuration = metrics.NewHistogram(
		metrics.HistogramOpts{
			Subsystem:                       translatorSubsystem,
			Name:                            "translation_duration_seconds",
			Help:                            "Translation duration",
			NativeHistogramBucketFactor:     1.1,
			NativeHistogramMaxBucketNumber:  100,
			NativeHistogramMinResetDuration: time.Hour,
		},
		[]string{translatorNameLabel},
	)
	translationsRunning = metrics.NewGauge(
		metrics.GaugeOpts{
			Subsystem: translatorSubsystem,
			Name:      "translations_running",
			Help:      "Number of translations currently running",
		},
		[]string{translatorNameLabel},
	)
	domainsPerListener = metrics.NewGauge(
		metrics.GaugeOpts{
			Subsystem: routingSubsystem,
			Name:      "domains",
			Help:      "Number of domains per listener",
		},
		[]string{"namespace", "gateway", "port"},
	)
)

// TranslatorMetricsRecorder defines the interface for recording translator metrics.
type TranslatorMetricsRecorder interface {
	TranslationStart() func(error)
}

// translatorMetrics records metrics for translator operations.
type translatorMetrics struct {
	translatorName      string
	translationsTotal   metrics.Counter
	translationDuration metrics.Histogram
	translationsRunning metrics.Gauge
}

// NewTranslatorMetricsRecorder creates a new recorder for translator metrics.
func NewTranslatorMetricsRecorder(translatorName string) TranslatorMetricsRecorder {
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
	m.translationsRunning.Add(1,
		metrics.Label{Name: translatorNameLabel, Value: m.translatorName})

	return func(err error) {
		duration := time.Since(start)

		m.translationDuration.Observe(duration.Seconds(),
			metrics.Label{Name: translatorNameLabel, Value: m.translatorName})

		result := "success"
		if err != nil {
			result = "error"
		}

		m.translationsTotal.Inc([]metrics.Label{
			{Name: translatorNameLabel, Value: m.translatorName},
			{Name: "result", Value: result},
		}...)

		m.translationsRunning.Sub(1,
			metrics.Label{Name: translatorNameLabel, Value: m.translatorName})
	}
}

// DomainsPerListenerMetricLabels is used as an argument to SetDomainPerListener
type DomainsPerListenerMetricLabels struct {
	Namespace   string
	GatewayName string
	Port        string
}

// toMetricsLabels converts DomainPerListenerLabels to a slice of metrics.Labels.
func (r DomainsPerListenerMetricLabels) toMetricsLabels() []metrics.Label {
	return []metrics.Label{
		{Name: "namespace", Value: r.Namespace},
		{Name: "gateway", Value: r.GatewayName},
		{Name: "port", Value: r.Port},
	}
}

// SetDomainsPerListener sets the number of domains per listener gauge.
func SetDomainsPerListener(labels DomainsPerListenerMetricLabels, domains int) {
	domainsPerListener.Set(float64(domains), labels.toMetricsLabels()...)
}

// GetTranslationsTotalMetric returns the translations counter.
// This is provided for testing purposes.
func GetTranslationsTotalMetric() metrics.Counter {
	return translationsTotal
}

// GetTranslationDurationMetric returns the translation duration histogram.
// This is provided for testing purposes.
func GetTranslationDurationMetric() metrics.Histogram {
	return translationDuration
}

// GetTranslationsRunningMetric returns the translations running gauge.
// This is provided for testing purposes.
func GetTranslationsRunningMetric() metrics.Gauge {
	return translationsRunning
}

// GetDomainsPerListenerMetric returns the domains per listener gauge.
// This is provided for testing purposes.
func GetDomainsPerListenerMetric() metrics.Gauge {
	return domainsPerListener
}
