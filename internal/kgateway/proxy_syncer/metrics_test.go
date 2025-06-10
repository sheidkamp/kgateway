package proxy_syncer_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	. "github.com/kgateway-dev/kgateway/v2/internal/kgateway/proxy_syncer"
	"github.com/kgateway-dev/kgateway/v2/pkg/metrics"
	"github.com/kgateway-dev/kgateway/v2/pkg/metrics/metricstest"
)

func setupTest() {
	ResetMetrics()
}

func TestNewStatusSyncRecorder(t *testing.T) {
	setupTest()

	syncerName := "test-syncer"
	m := NewStatusSyncMetricsRecorder(syncerName)

	finishFunc := m.StatusSyncStart()
	finishFunc(nil)
	m.SetResources(StatusSyncResourcesMetricLabels{Namespace: "default", Name: "test", Resource: "route"}, 5)

	expectedMetrics := []string{
		"kgateway_status_syncer_status_syncs_total",
		"kgateway_status_syncer_status_sync_duration_seconds",
		"kgateway_status_syncer_resources",
	}

	currentMetrics := metricstest.MustGatherMetrics(t)

	for _, expected := range expectedMetrics {
		currentMetrics.AssertMetricExists(expected)
	}
}

func TestStatusSyncStart_Success(t *testing.T) {
	setupTest()

	m := NewStatusSyncMetricsRecorder("test-syncer")

	finishFunc := m.StatusSyncStart()
	time.Sleep(10 * time.Millisecond)
	finishFunc(nil)

	currentMetrics := metricstest.MustGatherMetrics(t)

	currentMetrics.AssertMetricLabels("kgateway_status_syncer_status_syncs_total", []metrics.Label{
		{Name: "result", Value: "success"},
		{Name: "syncer", Value: "test-syncer"},
	})
	currentMetrics.AssertMetricCounterValue("kgateway_status_syncer_status_syncs_total", 1)

	currentMetrics.AssertMetricLabels("kgateway_status_syncer_status_sync_duration_seconds", []metrics.Label{
		{Name: "syncer", Value: "test-syncer"},
	})
	currentMetrics.AssertHistogramPopulated("kgateway_status_syncer_status_sync_duration_seconds")
}

func TesStatusSyncStart_Error(t *testing.T) {
	setupTest()

	m := NewStatusSyncMetricsRecorder("test-syncer")

	finishFunc := m.StatusSyncStart()
	finishFunc(assert.AnError)

	currentMetrics := metricstest.MustGatherMetrics(t)

	currentMetrics.AssertMetricLabels("kgateway_status_syncer_status_syncs_total", []metrics.Label{
		{Name: "result", Value: "error"},
		{Name: "syncer", Value: "test-syncer"},
	})
	currentMetrics.AssertMetricCounterValue("kgateway_status_syncer_status_syncs_total", 1)
	currentMetrics.AssertMetricNotExists("kgateway_status_syncer_status_sync_duration_seconds")
}

func TestStatusSyncResources(t *testing.T) {
	setupTest()

	m := NewStatusSyncMetricsRecorder("test-statusSync")

	// Test SetResources.
	m.SetResources(StatusSyncResourcesMetricLabels{Namespace: "default", Name: "test", Resource: "route"}, 5)
	m.SetResources(StatusSyncResourcesMetricLabels{Namespace: "kube-system", Name: "test", Resource: "gateway"}, 3)

	// Assert.True(t, found, "kgateway_status_syncer_resources metric not found")

	expectedLabels := [][]metrics.Label{
		{
			{Name: "name", Value: "test"},
			{Name: "namespace", Value: "default"},
			{Name: "resource", Value: "route"},
			{Name: "syncer", Value: "test-statusSync"},
		},
		{
			{Name: "name", Value: "test"},
			{Name: "namespace", Value: "kube-system"},
			{Name: "resource", Value: "gateway"},
			{Name: "syncer", Value: "test-statusSync"},
		},
	}

	currentMetrics := metricstest.MustGatherMetrics(t)

	currentMetrics.AssertMetricsLabels("kgateway_status_syncer_resources", expectedLabels)
	currentMetrics.AssertMetricGaugeValues("kgateway_status_syncer_resources", []float64{5, 3})

	// Test IncResources.
	m.IncResources(StatusSyncResourcesMetricLabels{Namespace: "default", Name: "test", Resource: "route"})

	currentMetrics = metricstest.MustGatherMetrics(t)
	currentMetrics.AssertMetricsLabels("kgateway_status_syncer_resources", expectedLabels)
	currentMetrics.AssertMetricGaugeValues("kgateway_status_syncer_resources", []float64{6, 3})

	// Test DecResources.
	m.DecResources(StatusSyncResourcesMetricLabels{Namespace: "default", Name: "test", Resource: "route"})

	currentMetrics = metricstest.MustGatherMetrics(t)
	currentMetrics.AssertMetricsLabels("kgateway_status_syncer_resources", expectedLabels)
	currentMetrics.AssertMetricGaugeValues("kgateway_status_syncer_resources", []float64{5, 3})

	// Test ResetResources.
	m.ResetResources("route")

	currentMetrics = metricstest.MustGatherMetrics(t)
	currentMetrics.AssertMetricsLabels("kgateway_status_syncer_resources", expectedLabels)
	currentMetrics.AssertMetricGaugeValues("kgateway_status_syncer_resources", []float64{0, 3})
}
