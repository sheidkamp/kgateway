package test

import (
	"testing"

	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

// Generic metric structs
type MetricLabel struct {
	Name  string
	Value string
}

type HistogramMetricOutput struct {
	SampleCount uint64
	SampleSum   float64
}

// Gathered metrics interface
type GatheredMetrics interface {
	AssertMetricsLabels(name string, expectedLabels [][]*MetricLabel)
	AssertMetricLabels(name string, expectedLabels []*MetricLabel)
	AssertMetricCounterValue(name string, expectedValue float64)
	AssertMetricGaugeValue(name string, expectedValue float64)
	AssertMetricGaugeValues(name string, expectedValues []float64)
	AssertMetricHistogramValue(name string, expectedValue *HistogramMetricOutput)
	AssertHistogramPopulated(name string)
	AssertMetricExists(name string)
	AssertMetricNotExists(name string)
}

func MustGatherMetrics(t *testing.T) GatheredMetrics {
	return mustGatherPrometheusMetrics(t)
}

var _ GatheredMetrics = &prometheusGatheredMetrics{}

// gathered metrics implementation for prometheus
type prometheusGatheredMetrics struct {
	metrics map[string][]*dto.Metric
	t       *testing.T
}

func mustGatherPrometheusMetrics(t *testing.T) GatheredMetrics {
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

func (g *prometheusGatheredMetrics) MustGetMetric(name string) *dto.Metric {
	m, ok := g.metrics[name]
	require.True(g.t, ok, "Metric %s not found", name)
	require.Equal(g.t, 1, len(m), "Expected 1 metric for %s", name)
	return m[0]
}

func (g *prometheusGatheredMetrics) MustGetMetrics(name string, expectedCount int) []*dto.Metric {
	m, ok := g.metrics[name]
	require.True(g.t, ok, "Metric %s not found", name)
	require.Equal(g.t, expectedCount, len(m), "Expected %d metrics for %s", expectedCount, name)
	return m
}

func (g *prometheusGatheredMetrics) AssertMetricObjLabels(metric *dto.Metric, expectedLabels []*MetricLabel) {
	assert.Equal(g.t, len(expectedLabels), len(metric.Label), "Expected %d labels, got %d", len(expectedLabels), len(metric.Label))
	for i, label := range expectedLabels {
		assert.Equal(g.t, label.Name, *metric.Label[i].Name, "Label %d name mismatch - expected %s, got %s", i, label.Name, *metric.Label[i].Name)
		assert.Equal(g.t, label.Value, *metric.Label[i].Value, "Label %d value mismatch - expected %s, got %s", i, label.Value, *metric.Label[i].Value)
	}
}

func (g *prometheusGatheredMetrics) AssertMetricLabels(name string, expectedLabels []*MetricLabel) {
	metric := g.MustGetMetric(name)

	g.AssertMetricObjLabels(metric, expectedLabels)
}

func (g *prometheusGatheredMetrics) AssertMetricsLabels(name string, expectedLabels [][]*MetricLabel) {
	metrics := g.MustGetMetrics(name, len(expectedLabels))
	for i, m := range metrics {
		g.AssertMetricObjLabels(m, expectedLabels[i])
	}
}

// Counter value methods
func (g *prometheusGatheredMetrics) AssertMetricCounterValue(name string, expectedValue float64) {
	metric := g.MustGetMetric(name)
	assert.Equal(g.t, expectedValue, metric.Counter.GetValue(), "Metric %s value mismatch - expected %f, got %f", name, expectedValue, metric.Counter.GetValue())
}

// Gauge value methods
func (g *prometheusGatheredMetrics) AssertMetricGaugeValue(name string, expectedValue float64) {
	metric := g.MustGetMetric(name)
	assert.Equal(g.t, expectedValue, metric.Gauge.GetValue(), "Metric %s value mismatch - expected %f, got %f", name, expectedValue, metric.Gauge.GetValue())
}

func (g *prometheusGatheredMetrics) AssertMetricGaugeValues(name string, expectedValues []float64) {
	metrics := g.MustGetMetrics(name, len(expectedValues))
	for i, m := range metrics {
		assert.Equal(g.t, expectedValues[i], m.Gauge.GetValue(), "Metric[%d] %s value mismatch - expected %f, got %f", i, name, expectedValues[i], m.Gauge.GetValue())
	}
}

// Histogram value methods
func (g *prometheusGatheredMetrics) AssertMetricHistogramValue(name string, expectedValue *HistogramMetricOutput) {
	metric := g.MustGetMetric(name)
	assert.Equal(g.t, expectedValue, &HistogramMetricOutput{
		SampleCount: *metric.Histogram.SampleCount,
		SampleSum:   *metric.Histogram.SampleSum,
	}, "Metric %s histogram value mismatch - expected %v, got %v", name, expectedValue, &HistogramMetricOutput{
		SampleCount: *metric.Histogram.SampleCount,
		SampleSum:   *metric.Histogram.SampleSum,
	})
}

func (g *prometheusGatheredMetrics) AssertHistogramPopulated(name string) {
	metric := g.MustGetMetric(name)
	assert.True(g.t, *metric.Histogram.SampleCount > 0, "Histogram %s is not populated", name)
	assert.True(g.t, *metric.Histogram.SampleSum > 0, "Histogram %s is not populated", name)
}

// Metric existence methods
func (g *prometheusGatheredMetrics) AssertMetricExists(name string) {
	_, ok := g.metrics[name]
	assert.True(g.t, ok, "Metric %s not found", name)
}

func (g *prometheusGatheredMetrics) AssertMetricNotExists(name string) {
	_, ok := g.metrics[name]
	assert.False(g.t, ok, "Metric %s found", name)
}
