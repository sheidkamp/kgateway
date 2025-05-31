package metrics

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

func TestNewTranslatorMetrics(t *testing.T) {
	setupTest()

	translatorName := "test-translator"
	m := NewTranslatorRecorder(translatorName)

	assert.IsType(t, &translatorMetrics{}, m)
	assert.Equal(t, translatorName, (m.(*translatorMetrics)).translatorName)

	// Use the metrics to generate some data.
	finishFunc := m.TranslationStart()
	finishFunc(nil)
	m.SetResourceCount("default", "route", 5)

	// Verify metrics are registered by checking if we can collect them.
	metricFamilies, err := metrics.Registry.Gather()
	require.NoError(t, err)

	expectedMetrics := []string{
		"kgateway_translator_translations_total",
		"kgateway_translator_translation_duration_seconds",
		"kgateway_translator_resource_count",
	}

	foundMetrics := make(map[string]bool)
	for _, mf := range metricFamilies {
		foundMetrics[*mf.Name] = true
	}

	for _, expected := range expectedMetrics {
		assert.True(t, foundMetrics[expected], "Expected metric %s not found", expected)
	}
}

func TestTranslationStart_Success(t *testing.T) {
	setupTest()

	m := NewTranslatorRecorder("test-translator")

	// Simulate successful translation.
	finishFunc := m.TranslationStart()
	time.Sleep(10 * time.Millisecond)
	finishFunc(nil)

	metricFamilies, err := metrics.Registry.Gather()
	require.NoError(t, err)

	var found bool

	for _, mf := range metricFamilies {
		if *mf.Name == "kgateway_translator_translations_total" {
			found = true
			assert.Equal(t, 1, len(mf.Metric))

			// Check labels and value.
			metric := mf.Metric[0]
			assert.Equal(t, 2, len(metric.Label))
			assert.Equal(t, "result", *metric.Label[0].Name)
			assert.Equal(t, "success", *metric.Label[0].Value)
			assert.Equal(t, "translator", *metric.Label[1].Name)
			assert.Equal(t, "test-translator", *metric.Label[1].Value)
			assert.Equal(t, float64(1), metric.Counter.GetValue())
		}
	}

	assert.True(t, found, "kgateway_translator_translations_total metric not found")

	// Check that duration was recorded (should be > 0).
	var durationFound bool

	for _, mf := range metricFamilies {
		if *mf.Name == "kgateway_translator_translation_duration_seconds" {
			durationFound = true
			assert.Equal(t, 1, len(mf.Metric))
			assert.True(t, *mf.Metric[0].Histogram.SampleCount > 0)
			assert.True(t, *mf.Metric[0].Histogram.SampleSum > 0)
		}
	}

	assert.True(t, durationFound, "kgateway_translator_translation_duration_seconds metric not found")
}

func TestTranslationStart_Error(t *testing.T) {
	setupTest()

	m := NewTranslatorRecorder("test-translator")

	// Simulate failed translation.
	finishFunc := m.TranslationStart()
	finishFunc(assert.AnError)

	metricFamilies, err := metrics.Registry.Gather()
	require.NoError(t, err)

	var found bool
	for _, mf := range metricFamilies {
		if *mf.Name == "kgateway_translator_translations_total" {
			found = true
			assert.Equal(t, 1, len(mf.Metric))

			// Check labels and value.
			metric := mf.Metric[0]
			assert.Equal(t, 2, len(metric.Label))
			assert.Equal(t, "result", *metric.Label[0].Name)
			assert.Equal(t, "error", *metric.Label[0].Value)
			assert.Equal(t, "translator", *metric.Label[1].Name)
			assert.Equal(t, "test-translator", *metric.Label[1].Value)
			assert.Equal(t, float64(1), metric.Counter.GetValue())
		}
	}

	assert.True(t, found, "kgateway_translator_translations_total metric not found")
}

func TestTranslatorResourceCount(t *testing.T) {
	setupTest()

	m := NewTranslatorRecorder("test-translator")

	// Test SetResourceCount.
	m.SetResourceCount("default", "route", 5)
	m.SetResourceCount("kube-system", "gateway", 3)

	metricFamilies, err := metrics.Registry.Gather()
	require.NoError(t, err)

	// Find the resource count metric.
	var found bool
	for _, mf := range metricFamilies {
		if *mf.Name == "kgateway_translator_resource_count" {
			found = true
			assert.Equal(t, 2, len(mf.Metric))

			resourceValues := make(map[string]map[string]float64)

			for _, metric := range mf.Metric {
				assert.Equal(t, 3, len(metric.Label))
				assert.Equal(t, "namespace", *metric.Label[0].Name)
				assert.Equal(t, "resource", *metric.Label[1].Name)
				assert.Equal(t, "translator", *metric.Label[2].Name)
				assert.Equal(t, "test-translator", *metric.Label[2].Value)

				if _, exists := resourceValues[*metric.Label[1].Value]; !exists {
					resourceValues[*metric.Label[1].Value] = make(map[string]float64)
				}

				resourceValues[*metric.Label[1].Value][*metric.Label[0].Value] = metric.Gauge.GetValue()
			}

			assert.Equal(t, float64(5), resourceValues["route"]["default"])
			assert.Equal(t, float64(3), resourceValues["gateway"]["kube-system"])
		}
	}

	assert.True(t, found, "kgateway_translator_resource_count metric not found")

	// Test IncResourceCount.
	m.IncResourceCount("default", "route")

	metricFamilies, err = metrics.Registry.Gather()
	require.NoError(t, err)

	for _, mf := range metricFamilies {
		if *mf.Name == "kgateway_translator_resource_count" {
			for _, metric := range mf.Metric {
				if len(metric.Label) > 0 && *metric.Label[0].Value == "default" {
					assert.Equal(t, float64(6), metric.Gauge.GetValue())
				}
			}
		}
	}

	// Test DecResourceCount.
	m.DecResourceCount("default", "route")

	metricFamilies, err = metrics.Registry.Gather()
	require.NoError(t, err)

	for _, mf := range metricFamilies {
		if *mf.Name == "kgateway_translator_resource_count" {
			for _, metric := range mf.Metric {
				if len(metric.Label) > 0 && *metric.Label[0].Value == "default" {
					assert.Equal(t, float64(5), metric.Gauge.GetValue())
				}
			}
		}
	}
}
