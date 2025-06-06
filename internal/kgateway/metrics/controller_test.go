package metrics_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	. "github.com/kgateway-dev/kgateway/v2/internal/kgateway/metrics"
)

func TestNewControllerMetrics(t *testing.T) {
	setupTest()

	controllerName := "test-controller"
	m := NewControllerRecorder(controllerName)

	finishFunc := m.ReconcileStart()
	finishFunc(nil)

	expectedMetrics := []string{
		"kgateway_controller_reconciliations_total",
		"kgateway_controller_reconcile_duration_seconds",
	}

	currentMetrics := mustGatherMetrics(t)
	for _, expected := range expectedMetrics {
		currentMetrics.assertMetricExists(expected)
	}
}

func TestReconcileStart_Success(t *testing.T) {
	setupTest()

	m := NewControllerRecorder("test-controller")

	finishFunc := m.ReconcileStart()
	time.Sleep(10 * time.Millisecond)
	finishFunc(nil)

	currentMetrics := mustGatherMetrics(t)

	currentMetrics.assertMetricLabels("kgateway_controller_reconciliations_total", []*metricLabel{
		{name: "controller", value: "test-controller"},
		{name: "result", value: "success"},
	})
	currentMetrics.assertMetricCounterValue("kgateway_controller_reconciliations_total", 1)

	currentMetrics.assertMetricLabels("kgateway_controller_reconcile_duration_seconds", []*metricLabel{
		{name: "controller", value: "test-controller"},
	})
	currentMetrics.assertHistogramPopulated("kgateway_controller_reconcile_duration_seconds")

}

func TestReconcileStart_Error(t *testing.T) {
	setupTest()

	m := NewControllerRecorder("test-controller")

	finishFunc := m.ReconcileStart()
	finishFunc(assert.AnError)

	currentMetrics := mustGatherMetrics(t)

	currentMetrics.assertMetricLabels("kgateway_controller_reconciliations_total", []*metricLabel{
		{name: "controller", value: "test-controller"},
		{name: "result", value: "error"},
	})
	currentMetrics.assertMetricCounterValue("kgateway_controller_reconciliations_total", 1)

	currentMetrics.assertMetricLabels("kgateway_controller_reconcile_duration_seconds", []*metricLabel{
		{name: "controller", value: "test-controller"},
	})
	currentMetrics.assertHistogramPopulated("kgateway_controller_reconcile_duration_seconds")

}

func mockNow() time.Time {
	return time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)
}
