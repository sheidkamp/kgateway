package metrics_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/metrics"

	. "github.com/kgateway-dev/kgateway/v2/internal/kgateway/metrics"
)

func TestNewTranslatorMetrics(t *testing.T) {
	setupTest()

	translatorName := "test-translator"
	m := NewTranslatorRecorder(translatorName)

	finishFunc := m.TranslationStart()
	finishFunc(nil)
	m.SetResources(TranslatorResourcesLabels{Namespace: "default", Name: "test", Resource: "route"}, 5)

	metricFamilies, err := metrics.Registry.Gather()
	require.NoError(t, err)

	expectedMetrics := []string{
		"kgateway_translator_translations_total",
		"kgateway_translator_translation_duration_seconds",
		"kgateway_translator_resources",
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

	finishFunc := m.TranslationStart()
	finishFunc(assert.AnError)

	metricFamilies, err := metrics.Registry.Gather()
	require.NoError(t, err)

	var found bool
	for _, mf := range metricFamilies {
		if *mf.Name == "kgateway_translator_translations_total" {
			found = true
			assert.Equal(t, 1, len(mf.Metric))

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

func TestTranslatorResources(t *testing.T) {
	setupTest()

	m := NewTranslatorRecorder("test-translator")

	// Test SetResources.
	m.SetResources(TranslatorResourcesLabels{Namespace: "default", Name: "test", Resource: "route"}, 5)
	m.SetResources(TranslatorResourcesLabels{Namespace: "kube-system", Name: "test", Resource: "gateway"}, 3)

	metricFamilies, err := metrics.Registry.Gather()
	require.NoError(t, err)

	var found bool
	for _, mf := range metricFamilies {
		if *mf.Name == "kgateway_translator_resources" {
			found = true
			assert.Equal(t, 2, len(mf.Metric))

			resourceValues := make(map[string]map[string]map[string]float64)

			for _, metric := range mf.Metric {
				assert.Equal(t, 4, len(metric.Label))
				assert.Equal(t, "name", *metric.Label[0].Name)
				assert.Equal(t, "namespace", *metric.Label[1].Name)
				assert.Equal(t, "resource", *metric.Label[2].Name)
				assert.Equal(t, "translator", *metric.Label[3].Name)

				if _, exists := resourceValues[*metric.Label[1].Value]; !exists {
					resourceValues[*metric.Label[1].Value] = make(map[string]map[string]float64)
				}

				if _, exists := resourceValues[*metric.Label[1].Value][*metric.Label[0].Value]; !exists {
					resourceValues[*metric.Label[1].Value][*metric.Label[0].Value] = make(map[string]float64)
				}

				resourceValues[*metric.Label[1].Value][*metric.Label[0].Value][*metric.Label[2].Value] = metric.Gauge.GetValue()
			}

			assert.Equal(t, float64(5), resourceValues["default"]["test"]["route"])
			assert.Equal(t, float64(3), resourceValues["kube-system"]["test"]["gateway"])
		}
	}

	assert.True(t, found, "kgateway_translator_resources metric not found")

	// Test IncResources.
	m.IncResources(TranslatorResourcesLabels{Namespace: "default", Name: "test", Resource: "route"})

	metricFamilies, err = metrics.Registry.Gather()
	require.NoError(t, err)

	for _, mf := range metricFamilies {
		if *mf.Name == "kgateway_translator_resources" {
			for _, metric := range mf.Metric {
				if len(metric.Label) > 0 && *metric.Label[0].Value == "default" {
					assert.Equal(t, float64(6), metric.Gauge.GetValue())
				}
			}
		}
	}

	// Test DecResources.
	m.DecResources(TranslatorResourcesLabels{Namespace: "default", Name: "test", Resource: "route"})

	metricFamilies, err = metrics.Registry.Gather()
	require.NoError(t, err)

	for _, mf := range metricFamilies {
		if *mf.Name == "kgateway_translator_resources" {
			for _, metric := range mf.Metric {
				if len(metric.Label) > 0 && *metric.Label[0].Value == "default" {
					assert.Equal(t, float64(5), metric.Gauge.GetValue())
				}
			}
		}
	}

	// Test ResetResources.
	m.ResetResources("test")

	metricFamilies, err = metrics.Registry.Gather()
	require.NoError(t, err)

	found = false
	for _, mf := range metricFamilies {
		if *mf.Name == "kgateway_translator_resources" {
			found = true
			for _, metric := range mf.Metric {
				if len(metric.Label) > 0 && *metric.Label[1].Value == "route" {
					assert.Equal(t, float64(0), metric.Gauge.GetValue())
				}
			}
		}
	}

	require.True(t, found, "kgateway_translator_resources metric not found after reset")
}
