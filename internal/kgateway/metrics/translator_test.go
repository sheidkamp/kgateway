package metrics_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	. "github.com/kgateway-dev/kgateway/v2/internal/kgateway/metrics"
)

func TestNewTranslatorMetrics(t *testing.T) {
	setupTest()

	translatorName := "test-translator"
	m := NewTranslatorRecorder(translatorName)

	finishFunc := m.TranslationStart()
	finishFunc(nil)

	expectedMetrics := []string{
		"kgateway_translator_translations_total",
		"kgateway_translator_translation_duration_seconds",
		"kgateway_translator_translations_running",
	}

	currentMetrics := mustGatherMetrics(t)
	for _, expected := range expectedMetrics {
		currentMetrics.mustGetMetric(t, expected)
	}
}

func assertTranslationsRunning(t *testing.T, currentMetrics gatheredMetrics, translatorName string, count int) {
	translationsRunning := currentMetrics.mustGetMetric(t, "kgateway_translator_translations_running")
	assertMetricLabels(t, translationsRunning, []*metricLabel{
		{name: "translator", value: translatorName},
	})
	assert.Equal(t, count, int(translationsRunning.GetGaugeValue()))
}

func TestTranslationStart_Success(t *testing.T) {
	setupTest()

	m := NewTranslatorRecorder("test-translator")

	// Start translation
	finishFunc := m.TranslationStart()
	time.Sleep(10 * time.Millisecond)

	// Check that the translations_running metric is 1
	currentMetrics := mustGatherMetrics(t)
	assertTranslationsRunning(t, currentMetrics, "test-translator", 1)

	// Finish translation
	finishFunc(nil)
	time.Sleep(10 * time.Millisecond)
	currentMetrics = mustGatherMetrics(t)

	// Check the translations_running metric
	assertTranslationsRunning(t, currentMetrics, "test-translator", 0)

	// Check the translations_total metric
	translationsTotal := currentMetrics.mustGetMetric(t, "kgateway_translator_translations_total")
	assertMetricLabels(t, translationsTotal, []*metricLabel{
		{name: "result", value: "success"},
		{name: "translator", value: "test-translator"},
	})
	assert.Equal(t, float64(1), translationsTotal.GetCounterValue())

	// Check the translation_duration_seconds metric
	translationDuration := currentMetrics.mustGetMetric(t, "kgateway_translator_translation_duration_seconds")
	assertMetricLabels(t, translationDuration, []*metricLabel{
		{name: "translator", value: "test-translator"},
	})
	assert.True(t, translationDuration.GetHistogramValue().sampleCount > 0)
	assert.True(t, translationDuration.GetHistogramValue().sampleSum > 0)

}

func TestTranslationStart_Error(t *testing.T) {
	setupTest()

	m := NewTranslatorRecorder("test-translator")

	finishFunc := m.TranslationStart()
	currentMetrics := mustGatherMetrics(t)
	assertTranslationsRunning(t, currentMetrics, "test-translator", 1)

	finishFunc(assert.AnError)
	currentMetrics = mustGatherMetrics(t)
	assertTranslationsRunning(t, currentMetrics, "test-translator", 0)

	translationsTotal := currentMetrics.mustGetMetric(t, "kgateway_translator_translations_total")
	assertMetricLabels(t, translationsTotal, []*metricLabel{
		{name: "result", value: "error"},
		{name: "translator", value: "test-translator"},
	})
	assert.Equal(t, float64(1), translationsTotal.GetCounterValue())

}
