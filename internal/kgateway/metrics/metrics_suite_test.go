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

type gatheredMetrics struct {
	metrics map[string][]*dto.Metric
	err     error
}

func (g *gatheredMetrics) mustGetMetric(t *testing.T, name string) *dto.Metric {
	m, ok := g.metrics[name]
	require.True(t, ok, "Metric %s not found", name)
	require.Equal(t, 1, len(m), "Expected 1 metric for %s", name)
	return m[0]
}

func mustGatherMetrics(t *testing.T) gatheredMetrics {
	gathered := gatheredMetrics{
		metrics: make(map[string][]*dto.Metric),
	}
	metricFamilies, err := metrics.Registry.Gather()
	require.NoError(t, err)

	for _, mf := range metricFamilies {
		gathered.metrics[*mf.Name] = mf.Metric
	}

	return gathered
}

func assertMetricLabels(t *testing.T, metric *dto.Metric, labels []*dto.LabelPair) {
	assert.Equal(t, len(labels), len(metric.Label), "Expected %d labels, got %d", len(labels), len(metric.Label))
	for i, label := range labels {
		assert.Equal(t, label.Name, metric.Label[i].Name, "Label %d name mismatch - expected %s, got %s", i, label.Name, *metric.Label[i].Name)
		assert.Equal(t, label.Value, metric.Label[i].Value, "Label %d value mismatch - expected %s, got %s", i, label.Value, *metric.Label[i].Value)
	}
}
