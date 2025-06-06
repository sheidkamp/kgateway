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
		currentMetrics.assertMetricExists(expected)
	}
}

func assertTranslationsRunning(currentMetrics gatheredMetrics, translatorName string, count int) {
	currentMetrics.assertMetricLabels("kgateway_translator_translations_running", []*metricLabel{
		{name: "translator", value: translatorName},
	})
	currentMetrics.assertMetricGaugeValue("kgateway_translator_translations_running", float64(count))
}

func TestTranslationStart_Success(t *testing.T) {
	setupTest()

	m := NewTranslatorRecorder("test-translator")

	// Start translation
	finishFunc := m.TranslationStart()
	time.Sleep(10 * time.Millisecond)

	// Check that the translations_running metric is 1
	currentMetrics := mustGatherMetrics(t)
	assertTranslationsRunning(currentMetrics, "test-translator", 1)

	// Finish translation
	finishFunc(nil)
	time.Sleep(10 * time.Millisecond)
	currentMetrics = mustGatherMetrics(t)

	// Check the translations_running metric
	assertTranslationsRunning(currentMetrics, "test-translator", 0)

	// Check the translations_total metric
	currentMetrics.assertMetricLabels("kgateway_translator_translations_total", []*metricLabel{
		{name: "result", value: "success"},
		{name: "translator", value: "test-translator"},
	})
	currentMetrics.assertMetricCounterValue("kgateway_translator_translations_total", 1)

	// Check the translation_duration_seconds metric
	currentMetrics.assertMetricLabels("kgateway_translator_translation_duration_seconds", []*metricLabel{
		{name: "translator", value: "test-translator"},
	})
	currentMetrics.assertHistogramPopulated("kgateway_translator_translation_duration_seconds")

}

func TestTranslationStart_Error(t *testing.T) {
	setupTest()

	m := NewTranslatorRecorder("test-translator")

	finishFunc := m.TranslationStart()
	currentMetrics := mustGatherMetrics(t)
	assertTranslationsRunning(currentMetrics, "test-translator", 1)

	finishFunc(assert.AnError)
	currentMetrics = mustGatherMetrics(t)
	assertTranslationsRunning(currentMetrics, "test-translator", 0)

	currentMetrics.assertMetricLabels("kgateway_translator_translations_total", []*metricLabel{
		{name: "result", value: "error"},
		{name: "translator", value: "test-translator"},
	})
	currentMetrics.assertMetricCounterValue("kgateway_translator_translations_total", 1)

}
