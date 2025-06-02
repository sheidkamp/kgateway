package metrics

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

func TestNewControllerMetrics(t *testing.T) {
	setupTest()

	controllerName := "test-controller"
	m := NewControllerRecorder(controllerName)

	assert.IsType(t, &controllerMetrics{}, m)
	assert.Equal(t, controllerName, (m.(*controllerMetrics)).controllerName)

	// Use the metrics to generate some data.
	finishFunc := m.ReconcileStart()
	finishFunc(nil)

	// Verify metrics are registered by checking if we can collect them.
	metricFamilies, err := metrics.Registry.Gather()
	require.NoError(t, err)

	expectedMetrics := []string{
		"kgateway_controller_reconciliations_total",
		"kgateway_controller_reconcile_duration_seconds",
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

	m := NewControllerRecorder("test-controller")

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

	m := NewControllerRecorder("test-controller")

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

func TestControllerStartups(t *testing.T) {
	setupTest()

	m := NewControllerRecorder("test-controller")
	m.IncStartups()

	metricFamilies, err := metrics.Registry.Gather()
	require.NoError(t, err)

	var found bool

	fmt.Println("Gathered metrics:", metricFamilies)

	for _, mf := range metricFamilies {
		fmt.Println(*mf.Name)
		if *mf.Name == "kgateway_controller_startups_total" {
			found = true
			assert.Equal(t, 1, len(mf.Metric))

			for _, metric := range mf.Metric {
				assert.Equal(t, 2, len(metric.Label))
				assert.Equal(t, "controller", *metric.Label[0].Name)
				assert.Equal(t, "test-controller", *metric.Label[0].Value)
				assert.Equal(t, "start_time", *metric.Label[1].Name)
			}
		}
	}

	assert.True(t, found, "kgateway_controller_startups_total metric not found")
}
