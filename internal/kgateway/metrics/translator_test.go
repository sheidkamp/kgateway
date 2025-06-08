package metrics_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	. "github.com/kgateway-dev/kgateway/v2/internal/kgateway/metrics"
	"github.com/kgateway-dev/kgateway/v2/pkg/metrics"
	"github.com/kgateway-dev/kgateway/v2/pkg/metrics/metricstest"
)

func TestNewTranslatorRecorder(t *testing.T) {
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

	currentMetrics := metricstest.MustGatherMetrics(t)
	for _, expected := range expectedMetrics {
		currentMetrics.AssertMetricExists(expected)
	}
}

func assertTranslationsRunning(currentMetrics metricstest.GatheredMetrics, translatorName string, count int) {
	currentMetrics.AssertMetricLabels("kgateway_translator_translations_running", []metrics.Label{
		{Name: "translator", Value: translatorName},
	})
	currentMetrics.AssertMetricGaugeValue("kgateway_translator_translations_running", float64(count))
}

func TestTranslationStart_Success(t *testing.T) {
	setupTest()

	m := NewTranslatorRecorder("test-translator")

	// Start translation
	finishFunc := m.TranslationStart()
	time.Sleep(10 * time.Millisecond)

	// Check that the translations_running metric is 1
	currentMetrics := metricstest.MustGatherMetrics(t)
	assertTranslationsRunning(currentMetrics, "test-translator", 1)

	// Finish translation
	finishFunc(nil)
	time.Sleep(10 * time.Millisecond)
	currentMetrics = metricstest.MustGatherMetrics(t)

	// Check the translations_running metric
	assertTranslationsRunning(currentMetrics, "test-translator", 0)

	// Check the translations_total metric
	currentMetrics.AssertMetricLabels("kgateway_translator_translations_total", []metrics.Label{
		{Name: "result", Value: "success"},
		{Name: "translator", Value: "test-translator"},
	})
	currentMetrics.AssertMetricCounterValue("kgateway_translator_translations_total", 1)

	// Check the translation_duration_seconds metric
	currentMetrics.AssertMetricLabels("kgateway_translator_translation_duration_seconds", []metrics.Label{
		{Name: "translator", Value: "test-translator"},
	})
	currentMetrics.AssertHistogramPopulated("kgateway_translator_translation_duration_seconds")
}

func TestTranslationStart_Error(t *testing.T) {
	setupTest()

	m := NewTranslatorRecorder("test-translator")

	finishFunc := m.TranslationStart()
	currentMetrics := metricstest.MustGatherMetrics(t)
	assertTranslationsRunning(currentMetrics, "test-translator", 1)

	finishFunc(assert.AnError)
	currentMetrics = metricstest.MustGatherMetrics(t)
	assertTranslationsRunning(currentMetrics, "test-translator", 0)

	currentMetrics.AssertMetricLabels("kgateway_translator_translations_total", []metrics.Label{
		{Name: "result", Value: "error"},
		{Name: "translator", Value: "test-translator"},
	})
	currentMetrics.AssertMetricCounterValue("kgateway_translator_translations_total", 1)
}
