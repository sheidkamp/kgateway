package metrics

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

func TestNewSnapshotMetrics(t *testing.T) {
	setupTest()

	snapshotName := "test-snapshot"
	m := NewSnapshotRecorder(snapshotName)

	assert.IsType(t, &snapshotMetrics{}, m)
	assert.Equal(t, snapshotName, (m.(*snapshotMetrics)).snapshotName)

	// Use the metrics to generate some data.
	finishFunc := m.SnapshotStart("default")
	finishFunc(nil)
	m.SetResourceCount("default", 5)

	// Verify metrics are registered by checking if we can collect them.
	metricFamilies, err := metrics.Registry.Gather()
	require.NoError(t, err)

	expectedMetrics := []string{
		"kgateway_snapshot_syncs_total",
		"kgateway_snapshot_sync_duration_seconds",
		"kgateway_snapshot_resource_count",
	}

	foundMetrics := make(map[string]bool)
	for _, mf := range metricFamilies {
		foundMetrics[*mf.Name] = true
	}

	for _, expected := range expectedMetrics {
		assert.True(t, foundMetrics[expected], "Expected metric %s not found", expected)
	}
}

func TestSnapshotStart_Success(t *testing.T) {
	setupTest()

	m := NewSnapshotRecorder("test-snapshot")

	// Simulate successful snapshot.
	finishFunc := m.SnapshotStart("default")
	time.Sleep(10 * time.Millisecond)
	finishFunc(nil)

	metricFamilies, err := metrics.Registry.Gather()
	require.NoError(t, err)

	var found bool

	for _, mf := range metricFamilies {
		if *mf.Name == "kgateway_snapshot_syncs_total" {
			found = true
			assert.Equal(t, 1, len(mf.Metric))

			// Check labels and value.
			metric := mf.Metric[0]
			assert.Equal(t, 3, len(metric.Label))
			assert.Equal(t, "proxy", *metric.Label[0].Name)
			assert.Equal(t, "default", *metric.Label[0].Value)
			assert.Equal(t, "result", *metric.Label[1].Name)
			assert.Equal(t, "success", *metric.Label[1].Value)
			assert.Equal(t, "snapshot", *metric.Label[2].Name)
			assert.Equal(t, "test-snapshot", *metric.Label[2].Value)
			assert.Equal(t, float64(1), metric.Counter.GetValue())
		}
	}

	assert.True(t, found, "kgateway_snapshot_syncs_total metric not found")

	// Check that duration was recorded (should be > 0).
	var durationFound bool

	for _, mf := range metricFamilies {
		if *mf.Name == "kgateway_snapshot_sync_duration_seconds" {
			durationFound = true
			assert.Equal(t, 1, len(mf.Metric))
			assert.True(t, *mf.Metric[0].Histogram.SampleCount > 0)
			assert.True(t, *mf.Metric[0].Histogram.SampleSum > 0)
		}
	}

	assert.True(t, durationFound, "kgateway_snapshot_sync_duration_seconds metric not found")
}

func TestSnapshotStart_Error(t *testing.T) {
	setupTest()

	m := NewSnapshotRecorder("test-snapshot")

	// Simulate failed snapshot.
	finishFunc := m.SnapshotStart("default")
	finishFunc(assert.AnError)

	metricFamilies, err := metrics.Registry.Gather()
	require.NoError(t, err)

	var found bool
	for _, mf := range metricFamilies {
		if *mf.Name == "kgateway_snapshot_syncs_total" {
			found = true
			assert.Equal(t, 1, len(mf.Metric))

			// Check labels and value.
			metric := mf.Metric[0]
			assert.Equal(t, 3, len(metric.Label))
			assert.Equal(t, "proxy", *metric.Label[0].Name)
			assert.Equal(t, "default", *metric.Label[0].Value)
			assert.Equal(t, "result", *metric.Label[1].Name)
			assert.Equal(t, "error", *metric.Label[1].Value)
			assert.Equal(t, "snapshot", *metric.Label[2].Name)
			assert.Equal(t, "test-snapshot", *metric.Label[2].Value)
			assert.Equal(t, float64(1), metric.Counter.GetValue())
		}
	}

	assert.True(t, found, "kgateway_snapshot_syncs_total metric not found")
}

func TestSnapshotResourceCount(t *testing.T) {
	setupTest()

	m := NewSnapshotRecorder("test-snapshot")

	// Test SetResourceCount.
	m.SetResourceCount("default", 5)
	m.SetResourceCount("kube-system", 3)

	metricFamilies, err := metrics.Registry.Gather()
	require.NoError(t, err)

	// Find the resource count metric.
	var found bool
	for _, mf := range metricFamilies {
		if *mf.Name == "kgateway_snapshot_resource_count" {
			found = true
			assert.Equal(t, 2, len(mf.Metric))

			resourceValues := make(map[string]float64)

			for _, metric := range mf.Metric {
				assert.Equal(t, 2, len(metric.Label))
				assert.Equal(t, "proxy", *metric.Label[0].Name)
				assert.Equal(t, "snapshot", *metric.Label[1].Name)
				assert.Equal(t, "test-snapshot", *metric.Label[1].Value)

				resourceValues[*metric.Label[0].Value] = metric.Gauge.GetValue()
			}

			assert.Equal(t, float64(5), resourceValues["default"])
			assert.Equal(t, float64(3), resourceValues["kube-system"])
		}
	}

	assert.True(t, found, "kgateway_snapshot_resource_count metric not found")

	// Test IncResourceCount.
	m.IncResourceCount("default")

	metricFamilies, err = metrics.Registry.Gather()
	require.NoError(t, err)

	for _, mf := range metricFamilies {
		if *mf.Name == "kgateway_snapshot_resource_count" {
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
		if *mf.Name == "kgateway_snapshot_resource_count" {
			for _, metric := range mf.Metric {
				if len(metric.Label) > 0 && *metric.Label[0].Value == "default" {
					assert.Equal(t, float64(5), metric.Gauge.GetValue())
				}
			}
		}
	}
}
