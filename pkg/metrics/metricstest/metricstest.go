// Package metricstest provides utilities for testing metrics.
package metricstest

import (
	"io"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/prometheus/client_golang/prometheus/testutil/promlint"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	crmetrics "sigs.k8s.io/controller-runtime/pkg/metrics"

	"github.com/kgateway-dev/kgateway/v2/pkg/metrics"
)

// HistogramMetricOutput is a struct to hold histogram metric output values.
type HistogramMetricOutput struct {
	SampleCount uint64
	SampleSum   float64
}

// Gathered metrics interface.
type GatheredMetrics interface {
	AssertMetricsLabels(name string, expectedLabels [][]metrics.Label)
	AssertMetricLabels(name string, expectedLabels []metrics.Label)
	AssertMetricCounterValue(name string, expectedValue float64)
	AssertMetricCounterValues(name string, expectedValues []float64)
	AssertMetricCounterValuesBetween(name string, expectedValues [][]float64)
	AssertMetricGaugeValue(name string, expectedValue float64)
	AssertMetricGaugeValues(name string, expectedValues []float64)
	AssertMetricHistogramValue(name string, expectedValue HistogramMetricOutput)
	AssertHistogramPopulated(name string)
	AssertMetricExists(name string)
	AssertMetricNotExists(name string)
}

// MustGatherMetrics gathers metrics and returns them as GatheredMetrics.
func MustGatherMetrics(t require.TestingT) GatheredMetrics {
	return MustGatherPrometheusMetrics(t)
}

var _ GatheredMetrics = &prometheusGatheredMetrics{}

// Gathered metrics implementation for prometheus metrics.
type prometheusGatheredMetrics struct {
	metrics map[string][]*dto.Metric
	t       require.TestingT
}

// MustGatherPrometheusMetrics gathers metrics from the registry and returns them.
func MustGatherPrometheusMetrics(t require.TestingT) GatheredMetrics {
	gathered := prometheusGatheredMetrics{
		metrics: make(map[string][]*dto.Metric),
		t:       t,
	}
	metricFamilies, err := crmetrics.Registry.Gather()
	require.NoError(t, err)

	for _, mf := range metricFamilies {
		metrics := make([]*dto.Metric, len(mf.GetMetric()))
		copy(metrics, mf.GetMetric())
		gathered.metrics[mf.GetName()] = metrics
	}

	return &gathered
}

// MustGetMetric retrieves a single metric by name, ensuring it exists and has exactly one instance.
func (g *prometheusGatheredMetrics) MustGetMetric(name string) *dto.Metric {
	m, ok := g.metrics[name]
	require.True(g.t, ok, "Metric %s not found", name)
	require.Equal(g.t, 1, len(m), "Expected 1 metric for %s", name)
	return m[0]
}

// MustGetMetrics retrieves multiple metrics by name, ensuring they exist and have the expected count.
func (g *prometheusGatheredMetrics) MustGetMetrics(name string, expectedCount int) []*dto.Metric {
	m, ok := g.metrics[name]
	require.True(g.t, ok, "Metric %s not found", name)
	require.Equal(g.t, expectedCount, len(m), "Expected %d metrics for %s", expectedCount, name)
	return m
}

// AssertMetricObjLabels asserts that a metric has the expected labels.
func (g *prometheusGatheredMetrics) AssertMetricObjLabels(metric *dto.Metric, expectedLabels []metrics.Label) {
	assert.Equal(g.t, len(expectedLabels), len(metric.GetLabel()), "Expected %d labels, got %d", len(expectedLabels), len(metric.GetLabel()))
	for i, label := range expectedLabels {
		assert.Equal(g.t, label.Name, metric.GetLabel()[i].GetName(), "Label %d name mismatch - expected %s, got %s", i, label.Name, metric.GetLabel()[i].GetName())
		assert.Equal(g.t, label.Value, metric.GetLabel()[i].GetValue(), "Label %d value mismatch - expected %s, got %s", i, label.Value, metric.GetLabel()[i].GetValue())
	}
}

// AssertMetricLabels asserts that a metric has the expected labels.
func (g *prometheusGatheredMetrics) AssertMetricLabels(name string, expectedLabels []metrics.Label) {
	metric := g.MustGetMetric(name)

	g.AssertMetricObjLabels(metric, expectedLabels)
}

// AssertMetricsLabels asserts that multiple metrics have the expected labels.
func (g *prometheusGatheredMetrics) AssertMetricsLabels(name string, expectedLabels [][]metrics.Label) {
	metrics := g.MustGetMetrics(name, len(expectedLabels))
	for i, m := range metrics {
		g.AssertMetricObjLabels(m, expectedLabels[i])
	}
}

// AssertMetricCounterValue asserts that a counter metric has the expected value.
func (g *prometheusGatheredMetrics) AssertMetricCounterValue(name string, expectedValue float64) {
	metric := g.MustGetMetric(name)
	assert.Equal(g.t, expectedValue, metric.GetCounter().GetValue(), "Metric %s value mismatch - expected %f, got %f", name, expectedValue, metric.GetCounter().GetValue())
}

// AssertMetricCounterValues asserts that a counter metric has the expected values for multiple instances.
func (g *prometheusGatheredMetrics) AssertMetricCounterValues(name string, expectedValues []float64) {
	metrics := g.MustGetMetrics(name, len(expectedValues))
	for i, m := range metrics {
		assert.Equal(g.t, expectedValues[i], m.GetCounter().GetValue(),
			"Metric[%d] %s value mismatch - expected %f, got %f",
			i, name, expectedValues[i], m.GetCounter().GetValue())
	}
}

// AssertMetricCounterValuesBetween asserts that a counter metric has the expected values for multiple instances.
func (g *prometheusGatheredMetrics) AssertMetricCounterValuesBetween(name string, expectedValues [][]float64) {
	metrics := g.MustGetMetrics(name, len(expectedValues))
	for i, m := range metrics {
		if len(expectedValues[i]) != 2 {
			assert.Fail(g.t, "Expected exactly two values for value range for metric %s at index %d, got %d",
				name, i, len(expectedValues[i]))
		}

		assert.GreaterOrEqual(g.t, m.GetCounter().GetValue(), expectedValues[i][0],
			"Metric[%d] %s value mismatch - expected greater than or equal to %f, got %f",
			i, name, expectedValues[i][0], m.GetCounter().GetValue())
		assert.LessOrEqual(g.t, m.GetCounter().GetValue(), expectedValues[i][1],
			"Metric[%d] %s value mismatch - expected less than or equal to %f, got %f",
			i, name, expectedValues[i][1], m.GetCounter().GetValue())
	}
}

// AssertMetricCounterValue asserts that a counter metric has the expected value.
func (g *prometheusGatheredMetrics) AssertMetricGaugeValue(name string, expectedValue float64) {
	metric := g.MustGetMetric(name)
	assert.Equal(g.t, expectedValue, metric.GetGauge().GetValue(), "Metric %s value mismatch - expected %f, got %f", name, expectedValue, metric.GetGauge().GetValue())
}

// AssertMetricGaugeValues asserts that a gauge metric has the expected values for multiple instances.
func (g *prometheusGatheredMetrics) AssertMetricGaugeValues(name string, expectedValues []float64) {
	metrics := g.MustGetMetrics(name, len(expectedValues))
	for i, m := range metrics {
		assert.Equal(g.t, expectedValues[i], m.GetGauge().GetValue(), "Metric[%d] %s value mismatch - expected %f, got %f", i, name, expectedValues[i], m.GetGauge().GetValue())
	}
}

// AssertMetricHistogramValue asserts that a histogram metric has the expected sample count and sum.
func (g *prometheusGatheredMetrics) AssertMetricHistogramValue(name string, expectedValue HistogramMetricOutput) {
	metric := g.MustGetMetric(name)
	assert.Equal(g.t, expectedValue, HistogramMetricOutput{
		SampleCount: metric.GetHistogram().GetSampleCount(),
		SampleSum:   metric.GetHistogram().GetSampleSum(),
	}, "Metric %s histogram value mismatch - expected %v, got %v", name, expectedValue, HistogramMetricOutput{
		SampleCount: metric.GetHistogram().GetSampleCount(),
		SampleSum:   metric.GetHistogram().GetSampleSum(),
	})
}

// AssertHistogramPopulated asserts that a histogram metric is populated (has non-zero sample count and sum).
func (g *prometheusGatheredMetrics) AssertHistogramPopulated(name string) {
	metric := g.MustGetMetric(name)
	assert.True(g.t, metric.GetHistogram().GetSampleCount() > 0, "Histogram %s is not populated", name)
	assert.True(g.t, metric.GetHistogram().GetSampleSum() > 0, "Histogram %s is not populated", name)
}

// AssertMetricExists asserts that a metric with the given name exists.
func (g *prometheusGatheredMetrics) AssertMetricExists(name string) {
	_, ok := g.metrics[name]
	assert.True(g.t, ok, "Metric %s not found", name)
}

// AssertMetricNotExists asserts that a metric with the given name does not exist.
func (g *prometheusGatheredMetrics) AssertMetricNotExists(name string) {
	_, ok := g.metrics[name]
	assert.False(g.t, ok, "Metric %s found", name)
}

// GatherAndLint gathers metrics and runs a linter on them.
func GatherAndLint(metricNames ...string) ([]promlint.Problem, error) {
	return testutil.GatherAndLint(crmetrics.Registry, metricNames...)
}

// GatherAndCompare gathers metrics and runs a linter on them.
func GatherAndCompare(expected io.Reader, metricNames ...string) error {
	return testutil.GatherAndCompare(crmetrics.Registry, expected, metricNames...)
}

// CollectAndCompare collects metrics from a collector and compares them against expected values.
func CollectAndCompare(c any, expected io.Reader, metricNames ...string) error {
	if err := testutil.CollectAndCompare(metrics.GetPromCollector(c), expected, metricNames...); err != nil {
		return err
	}

	return nil
}
