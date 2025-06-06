package metrics_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	. "github.com/kgateway-dev/kgateway/v2/internal/kgateway/metrics"
)

func TestNewStatusSyncRecorder(t *testing.T) {
	setupTest()

	syncerName := "test-syncer"
	m := NewStatusSyncRecorder(syncerName)

	finishFunc := m.StatusSyncStart()
	finishFunc(nil)
	m.SetResources(StatusSyncResourcesLabels{Namespace: "default", Name: "test", Resource: "route"}, 5)

	expectedMetrics := []string{
		"kgateway_status_syncer_status_syncs_total",
		"kgateway_status_syncer_status_sync_duration_seconds",
		"kgateway_status_syncer_resources",
	}

	currentMetrics := mustGatherMetrics(t)

	for _, expected := range expectedMetrics {
		currentMetrics.assertMetricExists(expected)
	}
}

func TestStatusSyncStart_Success(t *testing.T) {
	setupTest()

	m := NewStatusSyncRecorder("test-syncer")

	finishFunc := m.StatusSyncStart()
	time.Sleep(10 * time.Millisecond)
	finishFunc(nil)

	currentMetrics := mustGatherMetrics(t)

	currentMetrics.assertMetricLabels("kgateway_status_syncer_status_syncs_total", []*metricLabel{
		{name: "result", value: "success"},
		{name: "syncer", value: "test-syncer"},
	})
	currentMetrics.assertMetricCounterValue("kgateway_status_syncer_status_syncs_total", 1)

	currentMetrics.assertMetricLabels("kgateway_status_syncer_status_sync_duration_seconds", []*metricLabel{
		{name: "syncer", value: "test-syncer"},
	})
	currentMetrics.assertHistogramPopulated("kgateway_status_syncer_status_sync_duration_seconds")
}

func TesStatusSyncStart_Error(t *testing.T) {
	setupTest()

	m := NewStatusSyncRecorder("test-syncer")

	finishFunc := m.StatusSyncStart()
	finishFunc(assert.AnError)

	currentMetrics := mustGatherMetrics(t)

	currentMetrics.assertMetricLabels("kgateway_status_syncer_status_syncs_total", []*metricLabel{
		{name: "result", value: "error"},
		{name: "syncer", value: "test-syncer"},
	})
	currentMetrics.assertMetricCounterValue("kgateway_status_syncer_status_syncs_total", 1)
	currentMetrics.assertMetricNotExists("kgateway_status_syncer_status_sync_duration_seconds")
}

func TestStatusSyncResources(t *testing.T) {
	setupTest()

	m := NewStatusSyncRecorder("test-statusSync")

	// Test SetResources.
	m.SetResources(StatusSyncResourcesLabels{Namespace: "default", Name: "test", Resource: "route"}, 5)
	m.SetResources(StatusSyncResourcesLabels{Namespace: "kube-system", Name: "test", Resource: "gateway"}, 3)

	// assert.True(t, found, "kgateway_status_syncer_resources metric not found")

	expectedLabels := [][]*metricLabel{
		{
			{name: "name", value: "test"},
			{name: "namespace", value: "default"},
			{name: "resource", value: "route"},
			{name: "syncer", value: "test-statusSync"},
		},
		{
			{name: "name", value: "test"},
			{name: "namespace", value: "kube-system"},
			{name: "resource", value: "gateway"},
			{name: "syncer", value: "test-statusSync"},
		},
	}

	currentMetrics := mustGatherMetrics(t)

	currentMetrics.assertMetricsLabels("kgateway_status_syncer_resources", expectedLabels)
	currentMetrics.assertMetricGaugeValues("kgateway_status_syncer_resources", []float64{5, 3})

	// Test IncResources.
	m.IncResources(StatusSyncResourcesLabels{Namespace: "default", Name: "test", Resource: "route"})

	currentMetrics = mustGatherMetrics(t)
	currentMetrics.assertMetricsLabels("kgateway_status_syncer_resources", expectedLabels)
	currentMetrics.assertMetricGaugeValues("kgateway_status_syncer_resources", []float64{6, 3})

	// Test DecResources.
	m.DecResources(StatusSyncResourcesLabels{Namespace: "default", Name: "test", Resource: "route"})

	currentMetrics = mustGatherMetrics(t)
	currentMetrics.assertMetricsLabels("kgateway_status_syncer_resources", expectedLabels)
	currentMetrics.assertMetricGaugeValues("kgateway_status_syncer_resources", []float64{5, 3})

	// Test ResetResources.
	m.ResetResources("route")

	currentMetrics = mustGatherMetrics(t)
	currentMetrics.assertMetricsLabels("kgateway_status_syncer_resources", expectedLabels)
	currentMetrics.assertMetricGaugeValues("kgateway_status_syncer_resources", []float64{0, 3})
}
