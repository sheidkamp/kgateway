package metrics

import (
	"github.com/kgateway-dev/kgateway/v2/pkg/metrics"
)

const (
	routingSubsystem = "routing"
)

var (
	domainsPerListener = metrics.NewGauge(
		metrics.GaugeOpts{
			Namespace: metricsNamespace,
			Subsystem: routingSubsystem,
			Name:      "domains",
			Help:      "Number of domains per listener",
		},
		[]string{"namespace", "gatewayName", "port"},
	)
)

// RoutingRecorder is an interface for recording routing metrics.
type RoutingRecorder interface {
	SetDomainPerListener(labels DomainPerListenerLabels, domains int)
}

var _ RoutingRecorder = &routingMetrics{}

// routingMetrics implements the RoutingRecorder interface
type routingMetrics struct {
	domainsPerListener metrics.Gauge
}

// DomainPerListenerLabels is used as an argument to SetDomainPerListener
type DomainPerListenerLabels struct {
	Namespace   string
	GatewayName string
	Port        string
}

// toMetricsLabels converts DomainPerListenerLabels to a slice of metrics.Labels.
func (r DomainPerListenerLabels) toMetricsLabels() []metrics.Label {
	return []metrics.Label{
		{Name: "namespace", Value: r.Namespace},
		{Name: "gatewayName", Value: r.GatewayName},
		{Name: "port", Value: r.Port},
	}
}

// NewRoutingRecorder creates a new RoutingRecorder.
func NewRoutingRecorder() RoutingRecorder {
	return &routingMetrics{
		domainsPerListener: domainsPerListener,
	}
}

// SetDomainPerListener sets the number of domains per listener gauge.
func (m *routingMetrics) SetDomainPerListener(labels DomainPerListenerLabels, domains int) {
	m.domainsPerListener.Set(float64(domains), labels.toMetricsLabels()...)
}

// GetDomainsPerListener returns the domains per listener gauge.
// This is provided for testing purposes.
func GetDomainsPerListener() metrics.Gauge {
	return domainsPerListener
}
