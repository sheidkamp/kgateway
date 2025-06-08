package metrics_test

import (
	"testing"

	"github.com/kgateway-dev/kgateway/v2/pkg/metrics"
	"github.com/kgateway-dev/kgateway/v2/pkg/metrics/metricstest"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	crmetrics "sigs.k8s.io/controller-runtime/pkg/metrics"
)

func TestCounterInterface(t *testing.T) {
	opts := metrics.CounterOpts{
		Name: "test_counter",
		Help: "A test counter metric",
	}

	counter := metrics.NewCounter(opts, []string{"label1", "label2"})

	counter.Inc(metrics.Label{Name: "label1", Value: "value1"}, metrics.Label{Name: "label2", Value: "value2"})

	gathered := metricstest.MustGatherMetrics(t)
	gathered.AssertMetricCounterValue("test_counter", 1.0)
	gathered.AssertMetricLabels("test_counter", []metrics.Label{
		{Name: "label1", Value: "value1"},
		{Name: "label2", Value: "value2"},
	})

	counter.Add(5.0, metrics.Label{Name: "label1", Value: "value1"}, metrics.Label{Name: "label2", Value: "value2"})
	gathered = metricstest.MustGatherMetrics(t)
	gathered.AssertMetricCounterValue("test_counter", 6.0)

	counter.Reset()
	gathered = metricstest.MustGatherMetrics(t)
	gathered.AssertMetricNotExists("test_counter")

	crmetrics.Registry.Unregister(metrics.GetPromCollector(counter))
}

func TestCounterPartialLabels(t *testing.T) {
	opts := metrics.CounterOpts{
		Name: "test_counter_partial",
		Help: "A test counter metric with partial labels",
	}

	counter := metrics.NewCounter(opts, []string{"label1", "label2", "label3"})

	// Test with only some labels provided.
	counter.Inc(metrics.Label{Name: "label3", Value: "value3"}, metrics.Label{Name: "label1", Value: "value1"})

	gathered := metricstest.MustGatherMetrics(t)
	gathered.AssertMetricCounterValue("test_counter_partial", 1.0)
	gathered.AssertMetricLabels("test_counter_partial", []metrics.Label{
		{Name: "label1", Value: "value1"},
		{Name: "label2", Value: ""},
		{Name: "label3", Value: "value3"},
	})

	crmetrics.Registry.Unregister(metrics.GetPromCollector(counter))
}

func TestCounterNoLabels(t *testing.T) {
	opts := metrics.CounterOpts{
		Name: "test_counter_no_labels",
		Help: "A test counter metric with no labels",
	}

	counter := metrics.NewCounter(opts, []string{})

	counter.Inc()
	counter.Add(2.5)

	gathered := metricstest.MustGatherMetrics(t)
	gathered.AssertMetricCounterValue("test_counter_no_labels", 3.5)

	crmetrics.Registry.Unregister(metrics.GetPromCollector(counter))
}

func TestCounterRegistrationPanic(t *testing.T) {
	opts := metrics.CounterOpts{
		Name: "test_counter_duplicate",
		Help: "A test counter metric",
	}

	counter1 := metrics.NewCounter(opts, []string{})

	// Attempting to create a counter with the same name should panic.
	assert.Panics(t, func() {
		metrics.NewCounter(opts, []string{})
	})

	crmetrics.Registry.Unregister(metrics.GetPromCollector(counter1))
}

func TestHistogramInterface(t *testing.T) {
	opts := metrics.HistogramOpts{
		Name:    "test_histogram",
		Help:    "A test histogram metric",
		Buckets: prometheus.DefBuckets,
	}

	histogram := metrics.NewHistogram(opts, []string{"label1", "label2"})

	histogram.Observe(1.5, metrics.Label{Name: "label1", Value: "value1"}, metrics.Label{Name: "label2", Value: "value2"})
	histogram.Observe(2.5, metrics.Label{Name: "label1", Value: "value1"}, metrics.Label{Name: "label2", Value: "value2"})

	gathered := metricstest.MustGatherMetrics(t)
	gathered.AssertMetricHistogramValue("test_histogram", metricstest.HistogramMetricOutput{
		SampleCount: 2,
		SampleSum:   4.0,
	})
	gathered.AssertMetricLabels("test_histogram", []metrics.Label{
		{Name: "label1", Value: "value1"},
		{Name: "label2", Value: "value2"},
	})

	histogram.Reset()
	gathered = metricstest.MustGatherMetrics(t)
	gathered.AssertMetricNotExists("test_histogram")

	crmetrics.Registry.Unregister(metrics.GetPromCollector(histogram))
}

func TestHistogramPartialLabels(t *testing.T) {
	opts := metrics.HistogramOpts{
		Name:    "test_histogram_partial",
		Help:    "A test histogram metric with partial labels",
		Buckets: prometheus.DefBuckets,
	}

	histogram := metrics.NewHistogram(opts, []string{"label1", "label2", "label3"})

	// Test with only some labels provided.
	histogram.Observe(3.14, metrics.Label{Name: "label1", Value: "value1"}, metrics.Label{Name: "label3", Value: "value3"})

	gathered := metricstest.MustGatherMetrics(t)
	gathered.AssertMetricHistogramValue("test_histogram_partial", metricstest.HistogramMetricOutput{
		SampleCount: 1,
		SampleSum:   3.14,
	})
	gathered.AssertMetricLabels("test_histogram_partial", []metrics.Label{
		{Name: "label1", Value: "value1"},
		{Name: "label2", Value: ""},
		{Name: "label3", Value: "value3"},
	})

	crmetrics.Registry.Unregister(metrics.GetPromCollector(histogram))
}

func TestHistogramNoLabels(t *testing.T) {
	opts := metrics.HistogramOpts{
		Name:    "test_histogram_no_labels",
		Help:    "A test histogram metric with no labels",
		Buckets: []float64{0.1, 0.5, 1.0, 2.5, 5.0, 10.0},
	}

	histogram := metrics.NewHistogram(opts, []string{})

	histogram.Observe(0.5)
	histogram.Observe(1.5)
	histogram.Observe(7.0)

	gathered := metricstest.MustGatherMetrics(t)
	gathered.AssertMetricHistogramValue("test_histogram_no_labels", metricstest.HistogramMetricOutput{
		SampleCount: 3,
		SampleSum:   9.0,
	})

	crmetrics.Registry.Unregister(metrics.GetPromCollector(histogram))
}

func TestHistogramRegistrationPanic(t *testing.T) {
	opts := metrics.HistogramOpts{
		Name:    "test_histogram_duplicate",
		Help:    "A test histogram metric",
		Buckets: prometheus.DefBuckets,
	}

	histogram1 := metrics.NewHistogram(opts, []string{})

	// Attempting to create a histogram with the same name should panic.
	assert.Panics(t, func() {
		metrics.NewHistogram(opts, []string{})
	})

	crmetrics.Registry.Unregister(metrics.GetPromCollector(histogram1))
}

func TestGaugeInterface(t *testing.T) {
	opts := metrics.GaugeOpts{
		Name: "test_gauge",
		Help: "A test gauge metric",
	}

	gauge := metrics.NewGauge(opts, []string{"label1", "label2"})

	labels := []metrics.Label{
		{Name: "label1", Value: "value1"},
		{Name: "label2", Value: "value2"},
	}

	gauge.Set(10.0, labels...)

	gathered := metricstest.MustGatherMetrics(t)
	gathered.AssertMetricGaugeValue("test_gauge", 10.0)
	gathered.AssertMetricLabels("test_gauge", labels)

	gauge.Add(5.0, labels...)
	gathered = metricstest.MustGatherMetrics(t)
	gathered.AssertMetricGaugeValue("test_gauge", 15.0)

	gauge.Sub(3.0, labels...)
	gathered = metricstest.MustGatherMetrics(t)
	gathered.AssertMetricGaugeValue("test_gauge", 12.0)

	gauge.Reset()
	gathered = metricstest.MustGatherMetrics(t)
	gathered.AssertMetricNotExists("test_gauge")

	crmetrics.Registry.Unregister(metrics.GetPromCollector(gauge))
}

func TestGaugePartialLabels(t *testing.T) {
	opts := metrics.GaugeOpts{
		Name: "test_gauge_partial",
		Help: "A test gauge metric with partial labels",
	}

	gauge := metrics.NewGauge(opts, []string{"label1", "label2", "label3"})

	// Test with only some labels provided.
	gauge.Set(42.0, metrics.Label{Name: "label3", Value: "value3"}, metrics.Label{Name: "label1", Value: "value1"})

	gathered := metricstest.MustGatherMetrics(t)
	gathered.AssertMetricGaugeValue("test_gauge_partial", 42.0)
	gathered.AssertMetricLabels("test_gauge_partial", []metrics.Label{
		{Name: "label1", Value: "value1"},
		{Name: "label2", Value: ""},
		{Name: "label3", Value: "value3"},
	})

	crmetrics.Registry.Unregister(metrics.GetPromCollector(gauge))
}

func TestGaugeNoLabels(t *testing.T) {
	opts := metrics.GaugeOpts{
		Name: "test_gauge_no_labels",
		Help: "A test gauge metric with no labels",
	}

	gauge := metrics.NewGauge(opts, []string{})

	gauge.Set(100.0)
	gauge.Add(50.0)
	gauge.Sub(25.0)

	gathered := metricstest.MustGatherMetrics(t)
	gathered.AssertMetricGaugeValue("test_gauge_no_labels", 125.0)

	crmetrics.Registry.Unregister(metrics.GetPromCollector(gauge))
}

func TestGaugeRegistrationPanic(t *testing.T) {
	opts := metrics.GaugeOpts{
		Name: "test_gauge_duplicate",
		Help: "A test gauge metric",
	}

	gauge1 := metrics.NewGauge(opts, []string{})

	// Attempting to create a gauge with the same name should panic.
	assert.Panics(t, func() {
		metrics.NewGauge(opts, []string{})
	})

	crmetrics.Registry.Unregister(metrics.GetPromCollector(gauge1))
}

func TestGetPromCollector(t *testing.T) {
	counterOpts := metrics.CounterOpts{
		Name: "test_collector_counter",
		Help: "A test counter for collector testing",
	}
	counter := metrics.NewCounter(counterOpts, []string{})
	counterCollector := metrics.GetPromCollector(counter)
	require.NotNil(t, counterCollector)
	assert.IsType(t, &prometheus.CounterVec{}, counterCollector)

	histogramOpts := metrics.HistogramOpts{
		Name:    "test_collector_histogram",
		Help:    "A test histogram for collector testing",
		Buckets: prometheus.DefBuckets,
	}
	histogram := metrics.NewHistogram(histogramOpts, []string{})
	histogramCollector := metrics.GetPromCollector(histogram)
	require.NotNil(t, histogramCollector)
	assert.IsType(t, &prometheus.HistogramVec{}, histogramCollector)

	gaugeOpts := metrics.GaugeOpts{
		Name: "test_collector_gauge",
		Help: "A test gauge for collector testing",
	}
	gauge := metrics.NewGauge(gaugeOpts, []string{})
	gaugeCollector := metrics.GetPromCollector(gauge)
	require.NotNil(t, gaugeCollector)
	assert.IsType(t, &prometheus.GaugeVec{}, gaugeCollector)

	invalidCollector := metrics.GetPromCollector("invalid")
	assert.Nil(t, invalidCollector)

	crmetrics.Registry.Unregister(counterCollector)
	crmetrics.Registry.Unregister(histogramCollector)
	crmetrics.Registry.Unregister(gaugeCollector)
}

func TestValidateLabelsOrder(t *testing.T) {
	opts := metrics.CounterOpts{
		Name: "test_label_order",
		Help: "A test counter for label order testing",
	}

	counter := metrics.NewCounter(opts, []string{"z_label", "a_label", "m_label"})

	// Provide labels in different order than defined.
	counter.Inc(
		metrics.Label{Name: "a_label", Value: "a_value"},
		metrics.Label{Name: "z_label", Value: "z_value"},
		metrics.Label{Name: "m_label", Value: "m_value"},
	)

	gathered := metricstest.MustGatherMetrics(t)
	// Labels are provided to the metric in the order they are defined, and gathered in alphabetical order.
	gathered.AssertMetricLabels("test_label_order", []metrics.Label{
		{Name: "a_label", Value: "a_value"},
		{Name: "m_label", Value: "m_value"},
		{Name: "z_label", Value: "z_value"},
	})

	crmetrics.Registry.Unregister(metrics.GetPromCollector(counter))
}

func TestLabelsWithEmptyValues(t *testing.T) {
	opts := metrics.CounterOpts{
		Name: "test_empty_labels",
		Help: "A test counter for empty label testing",
	}

	counter := metrics.NewCounter(opts, []string{"label1", "label2", "label3"})

	counter.Inc(
		metrics.Label{Name: "label1", Value: ""},
		metrics.Label{Name: "label2", Value: "non_empty"},
		metrics.Label{Name: "label3", Value: ""},
	)

	gathered := metricstest.MustGatherMetrics(t)
	gathered.AssertMetricLabels("test_empty_labels", []metrics.Label{
		{Name: "label1", Value: ""},
		{Name: "label2", Value: "non_empty"},
		{Name: "label3", Value: ""},
	})

	crmetrics.Registry.Unregister(metrics.GetPromCollector(counter))
}

func TestMetricTypesInterfaces(t *testing.T) {
	var counter metrics.Counter
	var histogram metrics.Histogram
	var gauge metrics.Gauge

	counterOpts := metrics.CounterOpts{Name: "interface_counter", Help: "Test"}
	histogramOpts := metrics.HistogramOpts{Name: "interface_histogram", Help: "Test", Buckets: prometheus.DefBuckets}
	gaugeOpts := metrics.GaugeOpts{Name: "interface_gauge", Help: "Test"}

	counter = metrics.NewCounter(counterOpts, []string{})
	histogram = metrics.NewHistogram(histogramOpts, []string{})
	gauge = metrics.NewGauge(gaugeOpts, []string{})

	counter.Inc()
	counter.Add(1.0)
	counter.Reset()

	histogram.Observe(1.0)
	histogram.Reset()

	gauge.Set(1.0)
	gauge.Add(1.0)
	gauge.Sub(1.0)
	gauge.Reset()

	crmetrics.Registry.Unregister(metrics.GetPromCollector(counter))
	crmetrics.Registry.Unregister(metrics.GetPromCollector(histogram))
	crmetrics.Registry.Unregister(metrics.GetPromCollector(gauge))
}
