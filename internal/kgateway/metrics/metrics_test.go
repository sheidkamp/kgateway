package metrics_test

import (
	"testing"

	. "github.com/kgateway-dev/kgateway/v2/internal/kgateway/metrics"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

func TestMain(m *testing.M) {
	m.Run()
}

func setupTest() {
	ResetCollectionMetrics()
	ResetControllerMetrics()
	ResetTranslatorMetrics()
	ResetStatusSyncMetrics()
	ResetRoutingMetrics()
}

// Generic metric structs
type metricLabel struct {
	name  string
	value string
}

type histogramMetricOutput struct {
	sampleCount uint64
	sampleSum   float64
}

// Gathered metrics interface
type gatheredMetrics interface {
	assertMetricLabels(name string, expectedLabels []*metricLabel)
	assertMetricCounterValue(name string, expectedValue float64)
	assertMetricGaugeValue(name string, expectedValue float64)
	assertMetricHistogramValue(name string, expectedValue *histogramMetricOutput)
	assertHistogramPopulated(name string)
	assertMetricExists(name string)
}

var _ gatheredMetrics = &prometheusGatheredMetrics{}

// gathered metrics implementation for prometheus
type prometheusGatheredMetrics struct {
	metrics map[string][]*dto.Metric
	t       *testing.T
}

func (g *prometheusGatheredMetrics) mustGetMetric(name string) *dto.Metric {
	m, ok := g.metrics[name]
	require.True(g.t, ok, "Metric %s not found", name)
	require.Equal(g.t, 1, len(m), "Expected 1 metric for %s", name)
	return m[0]
}

func (g *prometheusGatheredMetrics) assertMetricLabels(name string, expectedLabels []*metricLabel) {
	metric := g.mustGetMetric(name)

	assert.Equal(g.t, len(expectedLabels), len(metric.Label), "Expected %d labels, got %d", len(expectedLabels), len(metric.Label))
	for i, label := range expectedLabels {
		assert.Equal(g.t, label.name, *metric.Label[i].Name, "Label %d name mismatch - expected %s, got %s", i, label.name, *metric.Label[i].Name)
		assert.Equal(g.t, label.value, *metric.Label[i].Value, "Label %d value mismatch - expected %s, got %s", i, label.value, *metric.Label[i].Value)
	}
}

func (g *prometheusGatheredMetrics) assertMetricCounterValue(name string, expectedValue float64) {
	metric := g.mustGetMetric(name)
	assert.Equal(g.t, expectedValue, metric.Counter.GetValue())
}

func (g *prometheusGatheredMetrics) assertMetricGaugeValue(name string, expectedValue float64) {
	metric := g.mustGetMetric(name)
	assert.Equal(g.t, expectedValue, metric.Gauge.GetValue())
}

func (g *prometheusGatheredMetrics) assertMetricHistogramValue(name string, expectedValue *histogramMetricOutput) {
	metric := g.mustGetMetric(name)
	assert.Equal(g.t, expectedValue, &histogramMetricOutput{
		sampleCount: *metric.Histogram.SampleCount,
		sampleSum:   *metric.Histogram.SampleSum,
	})
}

func (g *prometheusGatheredMetrics) assertHistogramPopulated(name string) {
	metric := g.mustGetMetric(name)
	assert.True(g.t, *metric.Histogram.SampleCount > 0, "Histogram %s is not populated", name)
	assert.True(g.t, *metric.Histogram.SampleSum > 0, "Histogram %s is not populated", name)
}

func (g *prometheusGatheredMetrics) assertMetricExists(name string) {
	_, ok := g.metrics[name]
	assert.True(g.t, ok, "Metric %s not found", name)
}

func mustGatherPrometheusMetrics(t *testing.T) gatheredMetrics {
	gathered := prometheusGatheredMetrics{
		metrics: make(map[string][]*dto.Metric),
		t:       t,
	}
	metricFamilies, err := metrics.Registry.Gather()
	require.NoError(t, err)

	for _, mf := range metricFamilies {
		metrics := make([]*dto.Metric, len(mf.Metric))
		for i, m := range mf.Metric {
			metrics[i] = m
		}
		gathered.metrics[*mf.Name] = metrics
	}

	return &gathered
}

func mustGatherMetrics(t *testing.T) gatheredMetrics {
	return mustGatherPrometheusMetrics(t)
}
