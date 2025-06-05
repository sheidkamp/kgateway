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

type metricLabel struct {
	name  string
	value string
}

type testMetric interface {
	GetLabels() []*metricLabel
	GetCounterValue() float64
	GetGaugeValue() float64
	GetHistogramValue() *histogramMetricOutput
}

type prometheusMetric struct {
	metric *dto.Metric
}

type histogramMetricOutput struct {
	sampleCount uint64
	sampleSum   float64
}

func (m *prometheusMetric) GetLabels() []*metricLabel {
	labels := make([]*metricLabel, len(m.metric.Label))
	for i, label := range m.metric.Label {
		labels[i] = &metricLabel{
			name:  *label.Name,
			value: *label.Value,
		}
	}
	return labels
}

func (m *prometheusMetric) GetGaugeValue() float64 {
	return m.metric.Gauge.GetValue()
}

func (m *prometheusMetric) GetHistogramValue() *histogramMetricOutput {
	return &histogramMetricOutput{
		sampleCount: *m.metric.Histogram.SampleCount,
		sampleSum:   *m.metric.Histogram.SampleSum,
	}
}

func (m *prometheusMetric) GetCounterValue() float64 {
	return m.metric.Counter.GetValue()
}

type gatheredMetrics struct {
	metrics map[string][]testMetric
}

func (g *gatheredMetrics) mustGetMetric(t *testing.T, name string) testMetric {
	m, ok := g.metrics[name]
	require.True(t, ok, "Metric %s not found", name)
	require.Equal(t, 1, len(m), "Expected 1 metric for %s", name)
	return m[0]
}

func mustGatherPrometheusMetrics(t *testing.T) gatheredMetrics {
	gathered := gatheredMetrics{
		metrics: make(map[string][]testMetric),
	}
	metricFamilies, err := metrics.Registry.Gather()
	require.NoError(t, err)

	for _, mf := range metricFamilies {
		metrics := make([]testMetric, len(mf.Metric))
		for i, m := range mf.Metric {
			metrics[i] = &prometheusMetric{metric: m}
		}
		gathered.metrics[*mf.Name] = metrics
	}

	return gathered
}

func mustGatherMetrics(t *testing.T) gatheredMetrics {
	return mustGatherPrometheusMetrics(t)
}

func assertMetricLabels(t *testing.T, metric testMetric, expectedLabels []*metricLabel) {
	labels := metric.GetLabels()
	assert.Equal(t, len(expectedLabels), len(labels), "Expected %d labels, got %d", len(expectedLabels), len(labels))
	for i, label := range expectedLabels {
		assert.Equal(t, label.name, labels[i].name, "Label %d name mismatch - expected %s, got %s", i, label.name, labels[i].name)
		assert.Equal(t, label.value, labels[i].value, "Label %d value mismatch - expected %s, got %s", i, label.value, labels[i].value)
	}
}
