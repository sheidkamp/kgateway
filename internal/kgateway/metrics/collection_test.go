package metrics_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	. "github.com/kgateway-dev/kgateway/v2/internal/kgateway/metrics"
)

func TestNewCollectionMetrics(t *testing.T) {
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

	currentMetrics := mustGatherMetrics(t)
	for _, expected := range expectedMetrics {
		currentMetrics.assertMetricExists(expected)
	}
}

func TestTransformStart_Success(t *testing.T) {
	setupTest()

	m := NewCollectionRecorder("test-collection")

	finishFunc := m.TransformStart()
	time.Sleep(10 * time.Millisecond)
	finishFunc(nil)

	currentMetrics := mustGatherMetrics(t)

	currentMetrics.assertMetricLabels("kgateway_collection_transforms_total", []*metricLabel{
		{name: "collection", value: "test-collection"},
		{name: "result", value: "success"},
	})
	currentMetrics.assertMetricCounterValue("kgateway_collection_transforms_total", 1)

	currentMetrics.assertMetricLabels("kgateway_collection_transform_duration_seconds", []*metricLabel{
		{name: "collection", value: "test-collection"},
	})
	currentMetrics.assertHistogramPopulated("kgateway_collection_transform_duration_seconds")
}

func TestTransformStart_Error(t *testing.T) {
	setupTest()

	m := NewCollectionRecorder("test-collection")

	finishFunc := m.TransformStart()
	finishFunc(assert.AnError)

	currentMetrics := mustGatherMetrics(t)

	currentMetrics.assertMetricLabels("kgateway_collection_transforms_total", []*metricLabel{
		{name: "collection", value: "test-collection"},
		{name: "result", value: "error"},
	})
	currentMetrics.assertMetricCounterValue("kgateway_collection_transforms_total", 1)

	currentMetrics.assertMetricLabels("kgateway_collection_transform_duration_seconds", []*metricLabel{
		{name: "collection", value: "test-collection"},
	})
}

func TestCollectionResources(t *testing.T) {
	setupTest()

	m := NewCollectionRecorder("test-collection")

	// Test SetResources.
	m.SetResources(CollectionResourcesLabels{Namespace: "default", Name: "test", Resource: "route"}, 5)
	m.SetResources(CollectionResourcesLabels{Namespace: "kube-system", Name: "test", Resource: "gateway"}, 3)

	expectedLabels := [][]*metricLabel{
		{
			{name: "collection", value: "test-collection"},
			{name: "name", value: "test"},
			{name: "namespace", value: "default"},
			{name: "resource", value: "route"},
		},
		{
			{name: "collection", value: "test-collection"},
			{name: "name", value: "test"},
			{name: "namespace", value: "kube-system"},
			{name: "resource", value: "gateway"},
		},
	}

	currentMetrics := mustGatherMetrics(t)

	currentMetrics.assertMetricsLabels("kgateway_collection_resources", expectedLabels)
	currentMetrics.assertMetricGaugeValues("kgateway_collection_resources", []float64{5, 3})

	// Test IncResources.
	m.IncResources(CollectionResourcesLabels{Namespace: "default", Name: "test", Resource: "route"})

	currentMetrics = mustGatherMetrics(t)
	currentMetrics.assertMetricsLabels("kgateway_collection_resources", expectedLabels)
	currentMetrics.assertMetricGaugeValues("kgateway_collection_resources", []float64{6, 3})

	// Test DecResources.
	m.DecResources(CollectionResourcesLabels{Namespace: "default", Name: "test", Resource: "route"})

	currentMetrics = mustGatherMetrics(t)
	currentMetrics.assertMetricsLabels("kgateway_collection_resources", expectedLabels)
	currentMetrics.assertMetricGaugeValues("kgateway_collection_resources", []float64{5, 3})

	// Test ResetResources.
	m.ResetResources("route")

	currentMetrics = mustGatherMetrics(t)
	currentMetrics.assertMetricsLabels("kgateway_collection_resources", expectedLabels)
	currentMetrics.assertMetricGaugeValues("kgateway_collection_resources", []float64{0, 3})
}
