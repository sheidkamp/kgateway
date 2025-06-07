package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	sigmetrics "sigs.k8s.io/controller-runtime/pkg/metrics"
)

type CounterVec interface {
	WithLabelValues(lvs ...string) Counter
}

type Counter interface {
	Inc()
}

type CounterOpts struct {
	Namespace MetricsNamespace
	Subsystem MetricsSubsystem
	Name      string
	Help      string
}

type GaugeVec interface {
	WithLabelValues(lvs ...string) Gauge
}

type Gauge interface {
	Set(value float64)
}

type GaugeOpts struct {
	Namespace MetricsNamespace
	Subsystem MetricsSubsystem
	Name      string
	Help      string
}

type HistogramVec interface {
	WithLabelValues(lvs ...string) Histogram
}

type Histogram interface {
	Observe(value float64)
}

type HistogramOpts struct {
	Namespace                       MetricsNamespace
	Subsystem                       MetricsSubsystem
	Name                            string
	Help                            string
	NativeHistogramBucketFactor     float64
	NativeHistogramMaxBucketNumber  int64
	NativeHistogramMinResetDuration time.Duration
}

type Metrics interface {
	RegisterMetrics(metrics []Collector)
	NewCounterVec(opts CounterOpts, labels []string) CounterVec
	NewGaugeVec(opts GaugeOpts, labels []string) GaugeVec
	NewHistogramVec(opts HistogramOpts, labels []string) HistogramVec
}

// Eventually become a factory to return different metrics implementations
// based on the configuration.
func NewMetrics() Metrics {
	return newPrometheusMetrics()
}

type prometheusMetrics struct {
	//registry *prometheus.Registry
}

func newPrometheusMetrics() Metrics {
	return &prometheusMetrics{}
}

type prometheusCounterVec struct {
	vec *prometheus.CounterVec
}

func (v *prometheusCounterVec) WithLabelValues(lvs ...string) Counter {
	return v.vec.WithLabelValues(lvs...)
}

func (v *prometheusCounterVec) Collect(ch chan<- prometheus.Metric) {
	v.vec.Collect(ch)
}

func (v *prometheusCounterVec) Describe(ch chan<- *prometheus.Desc) {
	v.vec.Describe(ch)
}

func (m *prometheusMetrics) NewCounterVec(opts CounterOpts, labels []string) CounterVec {
	return &prometheusCounterVec{
		vec: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: string(opts.Namespace),
			Subsystem: string(opts.Subsystem),
			Name:      opts.Name,
			Help:      opts.Help,
		}, labels),
	}
}

type prometheusHistogramVec struct {
	vec *prometheus.HistogramVec
}

func (v *prometheusHistogramVec) WithLabelValues(lvs ...string) Histogram {
	return v.vec.WithLabelValues(lvs...)
}

func (v *prometheusHistogramVec) Collect(ch chan<- prometheus.Metric) {
	v.vec.Collect(ch)
}

func (v *prometheusHistogramVec) Describe(ch chan<- *prometheus.Desc) {
	v.vec.Describe(ch)
}

func (m *prometheusMetrics) NewHistogramVec(opts HistogramOpts, labels []string) HistogramVec {

	return &prometheusHistogramVec{
		vec: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: string(opts.Namespace),
			Subsystem: string(opts.Subsystem),
			Name:      opts.Name,
			Help:      opts.Help,
		}, labels),
	}
}

// Prometheus GaugeVec implementation
type prometheusGaugeVec struct {
	vec *prometheus.GaugeVec
}

func (m *prometheusMetrics) NewGaugeVec(opts GaugeOpts, labels []string) GaugeVec {
	return &prometheusGaugeVec{
		vec: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: string(opts.Namespace),
			Subsystem: string(opts.Subsystem),
			Name:      opts.Name,
			Help:      opts.Help,
		}, labels),
	}
}

func (v *prometheusGaugeVec) WithLabelValues(lvs ...string) Gauge {
	return v.vec.WithLabelValues(lvs...)
}

func (v *prometheusGaugeVec) Collect(ch chan<- prometheus.Metric) {
	v.vec.Collect(ch)
}

func (v *prometheusGaugeVec) Describe(ch chan<- *prometheus.Desc) {
	v.vec.Describe(ch)
}

// Registry
type prometheusCollector struct {
	collector prometheus.Collector
}

// Hack to get the registry working for now
type Collector interface{}

func (m *prometheusMetrics) RegisterMetrics(metrics []Collector) {
	for _, metric := range metrics {
		if err := sigmetrics.Registry.Register(metric.(prometheus.Collector)); err != nil {
			panic(err)
		}
	}
}
