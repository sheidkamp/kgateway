package metrics_test

import (
	"testing"

	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/metrics"

	. "github.com/kgateway-dev/kgateway/v2/internal/kgateway/metrics"
)

func TestMain(m *testing.M) {
	m.Run()
}

func setupTest() {
	GetTransformDuration().Reset()
	GetTransformsTotal().Reset()
	GetCollectionResources().Reset()
	GetReconcileDuration().Reset()
	GetReconciliationsTotal().Reset()
	GetStatusSyncDuration().Reset()
	GetStatusSyncsTotal().Reset()
	GetStatusSyncResources().Reset()
	GetTranslationDuration().Reset()
	GetTranslationsTotal().Reset()
	GetDomainsPerListener().Reset()
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
	assertMetricsLabels(name string, expectedLabels [][]*metricLabel)
	assertMetricLabels(name string, expectedLabels []*metricLabel)
	assertMetricCounterValue(name string, expectedValue float64)
	assertMetricGaugeValue(name string, expectedValue float64)
	assertMetricGaugeValues(name string, expectedValues []float64)
	assertMetricHistogramValue(name string, expectedValue *histogramMetricOutput)
	assertHistogramPopulated(name string)
	assertMetricExists(name string)
	assertMetricNotExists(name string)
}

func mustGatherMetrics(t *testing.T) gatheredMetrics {
	return mustGatherPrometheusMetrics(t)
}

var _ gatheredMetrics = &prometheusGatheredMetrics{}

// gathered metrics implementation for prometheus
type prometheusGatheredMetrics struct {
	metrics map[string][]*dto.Metric
	t       *testing.T
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
		copy(metrics, mf.Metric)
		gathered.metrics[*mf.Name] = metrics
	}

	return &gathered
}

func (g *prometheusGatheredMetrics) mustGetMetric(name string) *dto.Metric {
	m, ok := g.metrics[name]
	require.True(g.t, ok, "Metric %s not found", name)
	require.Equal(g.t, 1, len(m), "Expected 1 metric for %s", name)
	return m[0]
}

func (g *prometheusGatheredMetrics) mustGetMetrics(name string, expectedCount int) []*dto.Metric {
	m, ok := g.metrics[name]
	require.True(g.t, ok, "Metric %s not found", name)
	require.Equal(g.t, expectedCount, len(m), "Expected %d metrics for %s", expectedCount, name)
	return m
}

func (g *prometheusGatheredMetrics) assertMetricObjLabels(metric *dto.Metric, expectedLabels []*metricLabel) {
	assert.Equal(g.t, len(expectedLabels), len(metric.Label), "Expected %d labels, got %d", len(expectedLabels), len(metric.Label))
	for i, label := range expectedLabels {
		assert.Equal(g.t, label.name, *metric.Label[i].Name, "Label %d name mismatch - expected %s, got %s", i, label.name, *metric.Label[i].Name)
		assert.Equal(g.t, label.value, *metric.Label[i].Value, "Label %d value mismatch - expected %s, got %s", i, label.value, *metric.Label[i].Value)
	}
}

func (g *prometheusGatheredMetrics) assertMetricLabels(name string, expectedLabels []*metricLabel) {
	metric := g.mustGetMetric(name)

	g.assertMetricObjLabels(metric, expectedLabels)
}

func (g *prometheusGatheredMetrics) assertMetricsLabels(name string, expectedLabels [][]*metricLabel) {
	metrics := g.mustGetMetrics(name, len(expectedLabels))
	for i, m := range metrics {
		g.assertMetricObjLabels(m, expectedLabels[i])
	}
}

// Counter value methods
func (g *prometheusGatheredMetrics) assertMetricCounterValue(name string, expectedValue float64) {
	metric := g.mustGetMetric(name)
	assert.Equal(g.t, expectedValue, metric.Counter.GetValue(), "Metric %s value mismatch - expected %f, got %f", name, expectedValue, metric.Counter.GetValue())
}

// Gauge value methods
func (g *prometheusGatheredMetrics) assertMetricGaugeValue(name string, expectedValue float64) {
	metric := g.mustGetMetric(name)
	assert.Equal(g.t, expectedValue, metric.Gauge.GetValue(), "Metric %s value mismatch - expected %f, got %f", name, expectedValue, metric.Gauge.GetValue())
}

func (g *prometheusGatheredMetrics) assertMetricGaugeValues(name string, expectedValues []float64) {
	metrics := g.mustGetMetrics(name, len(expectedValues))
	for i, m := range metrics {
		assert.Equal(g.t, expectedValues[i], m.Gauge.GetValue(), "Metric[%d] %s value mismatch - expected %f, got %f", i, name, expectedValues[i], m.Gauge.GetValue())
	}
}

// Histogram value methods
func (g *prometheusGatheredMetrics) assertMetricHistogramValue(name string, expectedValue *histogramMetricOutput) {
	metric := g.mustGetMetric(name)
	assert.Equal(g.t, expectedValue, &histogramMetricOutput{
		sampleCount: *metric.Histogram.SampleCount,
		sampleSum:   *metric.Histogram.SampleSum,
	}, "Metric %s histogram value mismatch - expected %v, got %v", name, expectedValue, &histogramMetricOutput{
		sampleCount: *metric.Histogram.SampleCount,
		sampleSum:   *metric.Histogram.SampleSum,
	})
}

func (g *prometheusGatheredMetrics) assertHistogramPopulated(name string) {
	metric := g.mustGetMetric(name)
	assert.True(g.t, *metric.Histogram.SampleCount > 0, "Histogram %s is not populated", name)
	assert.True(g.t, *metric.Histogram.SampleSum > 0, "Histogram %s is not populated", name)
}

// Metric existence methods
func (g *prometheusGatheredMetrics) assertMetricExists(name string) {
	_, ok := g.metrics[name]
	assert.True(g.t, ok, "Metric %s not found", name)
}

func (g *prometheusGatheredMetrics) assertMetricNotExists(name string) {
	_, ok := g.metrics[name]
	assert.False(g.t, ok, "Metric %s found", name)
}
