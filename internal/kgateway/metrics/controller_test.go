package metrics

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

func TestNewControllerMetrics(t *testing.T) {
	setupTest()

	controllerName := "test-controller"
	m := NewControllerMetrics(controllerName)

	assert.Equal(t, controllerName, m.controllerName)

	// Use the metrics to generate some data.
	finishFunc := m.ReconcileStart()
	finishFunc(nil)
	m.SetResourceCount("default", 5)

	// Verify metrics are registered by checking if we can collect them.
	metricFamilies, err := metrics.Registry.Gather()
	require.NoError(t, err)

	expectedMetrics := []string{
		"kgateway_controller_reconciliations_total",
		"kgateway_controller_reconcile_duration_seconds",
		"kgateway_controller_resources_managed",
	}

	foundMetrics := make(map[string]bool)
	for _, mf := range metricFamilies {
		foundMetrics[*mf.Name] = true
	}

	for _, expected := range expectedMetrics {
		assert.True(t, foundMetrics[expected], "Expected metric %s not found", expected)
	}
}

func TestReconcileStart_Success(t *testing.T) {
	setupTest()

	m := NewControllerMetrics("test-controller")

	// Simulate successful reconcile.
	finishFunc := m.ReconcileStart()
	time.Sleep(10 * time.Millisecond)
	finishFunc(nil)

	metricFamilies, err := metrics.Registry.Gather()
	require.NoError(t, err)

	var found bool

	for _, mf := range metricFamilies {
		if *mf.Name == "kgateway_controller_reconciliations_total" {
			found = true
			assert.Equal(t, 1, len(mf.Metric))

			// Check labels and value.
			metric := mf.Metric[0]
			assert.Equal(t, 2, len(metric.Label))
			assert.Equal(t, "controller", *metric.Label[0].Name)
			assert.Equal(t, "test-controller", *metric.Label[0].Value)
			assert.Equal(t, "result", *metric.Label[1].Name)
			assert.Equal(t, "success", *metric.Label[1].Value)
			assert.Equal(t, float64(1), metric.Counter.GetValue())
		}
	}

	assert.True(t, found, "kgateway_controller_reconciliations_total metric not found")

	// Check that duration was recorded (should be > 0).
	var durationFound bool

	for _, mf := range metricFamilies {
		if *mf.Name == "kgateway_controller_reconcile_duration_seconds" {
			durationFound = true
			assert.Equal(t, 1, len(mf.Metric))
			assert.True(t, *mf.Metric[0].Histogram.SampleCount > 0)
			assert.True(t, *mf.Metric[0].Histogram.SampleSum > 0)
		}
	}

	assert.True(t, durationFound, "kgateway_controller_reconcile_duration_seconds metric not found")
}

func TestReconcileStart_Error(t *testing.T) {
	setupTest()

	m := NewControllerMetrics("test-controller")

	// Simulate failed reconcile.
	finishFunc := m.ReconcileStart()
	finishFunc(assert.AnError)

	metricFamilies, err := metrics.Registry.Gather()
	require.NoError(t, err)

	var found bool
	for _, mf := range metricFamilies {
		if *mf.Name == "kgateway_controller_reconciliations_total" {
			found = true
			assert.Equal(t, 1, len(mf.Metric))

			// Check labels and value.
			metric := mf.Metric[0]
			assert.Equal(t, 2, len(metric.Label))
			assert.Equal(t, "controller", *metric.Label[0].Name)
			assert.Equal(t, "test-controller", *metric.Label[0].Value)
			assert.Equal(t, "result", *metric.Label[1].Name)
			assert.Equal(t, "error", *metric.Label[1].Value)
			assert.Equal(t, float64(1), metric.Counter.GetValue())
		}
	}

	assert.True(t, found, "kgateway_controller_reconciliations_total metric not found")
}

func TestControllerResourceCount(t *testing.T) {
	setupTest()

	m := NewControllerMetrics("test-controller")

	// Test SetResourceCount.
	m.SetResourceCount("default", 5)
	m.SetResourceCount("kube-system", 3)

	metricFamilies, err := metrics.Registry.Gather()
	require.NoError(t, err)

	// Find the resource count metric.
	var found bool
	for _, mf := range metricFamilies {
		if *mf.Name == "kgateway_controller_resources_managed" {
			found = true
			assert.Equal(t, 2, len(mf.Metric))

			namespaceValues := make(map[string]float64)
			for _, metric := range mf.Metric {
				assert.Equal(t, 2, len(metric.Label))
				assert.Equal(t, "controller", *metric.Label[0].Name)
				assert.Equal(t, "test-controller", *metric.Label[0].Value)
				assert.Equal(t, "namespace", *metric.Label[1].Name)
				namespaceValues[*metric.Label[1].Value] = metric.Gauge.GetValue()
			}

			assert.Equal(t, float64(5), namespaceValues["default"])
			assert.Equal(t, float64(3), namespaceValues["kube-system"])
		}
	}

	assert.True(t, found, "kgateway_controller_resources_managed metric not found")

	// Test IncResourceCount.
	m.IncResourceCount("default")

	metricFamilies, err = metrics.Registry.Gather()
	require.NoError(t, err)

	for _, mf := range metricFamilies {
		if *mf.Name == "kgateway_controller_resources_managed" {
			for _, metric := range mf.Metric {
				if len(metric.Label) > 0 && *metric.Label[0].Value == "default" {
					assert.Equal(t, float64(6), metric.Gauge.GetValue())
				}
			}
		}
	}

	// Test DecResourceCount.
	m.DecResourceCount("default")

	metricFamilies, err = metrics.Registry.Gather()
	require.NoError(t, err)

	for _, mf := range metricFamilies {
		if *mf.Name == "kgateway_controller_resources_managed" {
			for _, metric := range mf.Metric {
				if len(metric.Label) > 0 && *metric.Label[0].Value == "default" {
					assert.Equal(t, float64(5), metric.Gauge.GetValue())
				}
			}
		}
	}

	// Test ResetResourceCounts.
	m.ResetResourceCounts()

	metricFamilies, err = metrics.Registry.Gather()
	require.NoError(t, err)

	for _, mf := range metricFamilies {
		if *mf.Name == "kgateway_controller_resources_managed" {
			assert.Equal(t, 0, len(mf.Metric), "Expected no metrics after reset")
		}
	}
}
