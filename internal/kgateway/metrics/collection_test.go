package metrics_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	. "github.com/kgateway-dev/kgateway/v2/internal/kgateway/metrics"
	"github.com/kgateway-dev/kgateway/v2/pkg/metrics"
	"github.com/kgateway-dev/kgateway/v2/pkg/metrics/metricstest"
)

func TestNewCollectionRecorder(t *testing.T) {
	setupTest()

	collectionName := "test-collection"
	m := NewCollectionRecorder(collectionName)

	finishFunc := m.TransformStart()
	finishFunc(nil)
	m.SetResources(CollectionResourcesLabels{Namespace: "default", Name: "test", Resource: "route"}, 5)

	expectedMetrics := []string{
		"kgateway_collection_transforms_total",
		"kgateway_collection_transform_duration_seconds",
		"kgateway_collection_resources",
	}

	currentMetrics := metricstest.MustGatherMetrics(t)
	for _, expected := range expectedMetrics {
		currentMetrics.AssertMetricExists(expected)
	}
}

func TestTransformStart_Success(t *testing.T) {
	setupTest()

	m := NewCollectionRecorder("test-collection")

	finishFunc := m.TransformStart()
	time.Sleep(10 * time.Millisecond)
	finishFunc(nil)

	currentMetrics := metricstest.MustGatherMetrics(t)

	currentMetrics.AssertMetricLabels("kgateway_collection_transforms_total", []metrics.Label{
		{Name: "collection", Value: "test-collection"},
		{Name: "result", Value: "success"},
	})
	currentMetrics.AssertMetricCounterValue("kgateway_collection_transforms_total", 1)

	currentMetrics.AssertMetricLabels("kgateway_collection_transform_duration_seconds", []metrics.Label{
		{Name: "collection", Value: "test-collection"},
	})
	currentMetrics.AssertHistogramPopulated("kgateway_collection_transform_duration_seconds")
}

func TestTransformStart_Error(t *testing.T) {
	setupTest()

	m := NewCollectionRecorder("test-collection")

	finishFunc := m.TransformStart()
	finishFunc(assert.AnError)

	currentMetrics := metricstest.MustGatherMetrics(t)

	currentMetrics.AssertMetricLabels("kgateway_collection_transforms_total", []metrics.Label{
		{Name: "collection", Value: "test-collection"},
		{Name: "result", Value: "error"},
	})
	currentMetrics.AssertMetricCounterValue("kgateway_collection_transforms_total", 1)

	currentMetrics.AssertMetricLabels("kgateway_collection_transform_duration_seconds", []metrics.Label{
		{Name: "collection", Value: "test-collection"},
	})
	currentMetrics.AssertHistogramPopulated("kgateway_collection_transform_duration_seconds")
}

func TestCollectionResources(t *testing.T) {
	setupTest()

	m := NewCollectionRecorder("test-collection")

	// Test SetResources.
	m.SetResources(CollectionResourcesLabels{Namespace: "default", Name: "test", Resource: "route"}, 5)
	m.SetResources(CollectionResourcesLabels{Namespace: "kube-system", Name: "test", Resource: "gateway"}, 3)

	expectedLabels := [][]metrics.Label{{
		{Name: "collection", Value: "test-collection"},
		{Name: "name", Value: "test"},
		{Name: "namespace", Value: "default"},
		{Name: "resource", Value: "route"},
	}, {
		{Name: "collection", Value: "test-collection"},
		{Name: "name", Value: "test"},
		{Name: "namespace", Value: "kube-system"},
		{Name: "resource", Value: "gateway"},
	}}

	currentMetrics := metricstest.MustGatherMetrics(t)

	currentMetrics.AssertMetricsLabels("kgateway_collection_resources", expectedLabels)
	currentMetrics.AssertMetricGaugeValues("kgateway_collection_resources", []float64{5, 3})

	// Test IncResources.
	m.IncResources(CollectionResourcesLabels{Namespace: "default", Name: "test", Resource: "route"})

	currentMetrics = metricstest.MustGatherMetrics(t)
	currentMetrics.AssertMetricsLabels("kgateway_collection_resources", expectedLabels)
	currentMetrics.AssertMetricGaugeValues("kgateway_collection_resources", []float64{6, 3})

	// Test DecResources.
	m.DecResources(CollectionResourcesLabels{Namespace: "default", Name: "test", Resource: "route"})

	currentMetrics = metricstest.MustGatherMetrics(t)
	currentMetrics.AssertMetricsLabels("kgateway_collection_resources", expectedLabels)
	currentMetrics.AssertMetricGaugeValues("kgateway_collection_resources", []float64{5, 3})

	// Test ResetResources.
	m.ResetResources("route")

	currentMetrics = metricstest.MustGatherMetrics(t)
	currentMetrics.AssertMetricsLabels("kgateway_collection_resources", expectedLabels)
	currentMetrics.AssertMetricGaugeValues("kgateway_collection_resources", []float64{0, 3})
}
