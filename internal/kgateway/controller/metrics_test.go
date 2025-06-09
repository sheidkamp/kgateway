package controller_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	. "github.com/kgateway-dev/kgateway/v2/internal/kgateway/controller"
	metricstest "github.com/kgateway-dev/kgateway/v2/internal/kgateway/metrics/test"
)

func setupTestMetrics() {
	GetReconciliationsTotal().Reset()
	GetReconcileDuration().Reset()
}

func TestNewControllerRecorder(t *testing.T) {
	setupTestMetrics()

	controllerName := "test-controller"
	m := NewControllerRecorder(controllerName)

	finishFunc := m.ReconcileStart()
	finishFunc(nil)

	expectedMetrics := []string{
		"kgateway_controller_reconciliations_total",
		"kgateway_controller_reconcile_duration_seconds",
	}

	currentMetrics := metricstest.MustGatherMetrics(t)
	for _, expected := range expectedMetrics {
		currentMetrics.AssertMetricExists(expected)
	}
}

func TestReconcileStart_Success(t *testing.T) {
	setupTestMetrics()

	m := NewControllerRecorder("test-controller")

	finishFunc := m.ReconcileStart()
	time.Sleep(10 * time.Millisecond)
	finishFunc(nil)

	currentMetrics := metricstest.MustGatherMetrics(t)

	currentMetrics.AssertMetricLabels("kgateway_controller_reconciliations_total", []*metricstest.MetricLabel{
		{Name: "controller", Value: "test-controller"},
		{Name: "result", Value: "success"},
	})
	currentMetrics.AssertMetricCounterValue("kgateway_controller_reconciliations_total", 1)

	currentMetrics.AssertMetricLabels("kgateway_controller_reconcile_duration_seconds", []*metricstest.MetricLabel{
		{Name: "controller", Value: "test-controller"},
	})
	currentMetrics.AssertHistogramPopulated("kgateway_controller_reconcile_duration_seconds")
}

func TestReconcileStart_Error(t *testing.T) {
	setupTestMetrics()

	m := NewControllerRecorder("test-controller")

	finishFunc := m.ReconcileStart()
	finishFunc(assert.AnError)

	currentMetrics := metricstest.MustGatherMetrics(t)

	currentMetrics.AssertMetricLabels("kgateway_controller_reconciliations_total", []*metricstest.MetricLabel{
		{Name: "controller", Value: "test-controller"},
		{Name: "result", Value: "error"},
	})
	currentMetrics.AssertMetricCounterValue("kgateway_controller_reconciliations_total", 1)

	currentMetrics.AssertMetricLabels("kgateway_controller_reconcile_duration_seconds", []*metricstest.MetricLabel{
		{Name: "controller", Value: "test-controller"},
	})
	currentMetrics.AssertHistogramPopulated("kgateway_controller_reconcile_duration_seconds")
}
