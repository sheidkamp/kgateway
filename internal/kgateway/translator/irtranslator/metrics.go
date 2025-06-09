package irtranslator

import (
	kgatewaymetrics "github.com/kgateway-dev/kgateway/v2/internal/kgateway/metrics"
	"github.com/kgateway-dev/kgateway/v2/pkg/metrics"
)

const (
	routingSubsystem = "routing"
)

var (
	domainsPerListener = metrics.NewGauge(
		metrics.GaugeOpts{
			Namespace: kgatewaymetrics.MetricsNamespace,
			Subsystem: routingSubsystem,
			Name:      "domains",
			Help:      "Number of domains per listener",
		},
		[]string{"namespace", "gatewayName", "port"},
	)
)

// routingMetrics implements the RoutingRecorder interface

// domainPerListenerLabels is used as an argument to SetDomainPerListener
type domainPerListenerLabels struct {
	Namespace   string
	GatewayName string
	Port        string
}

// toMetricsLabels converts DomainPerListenerLabels to a slice of metrics.Labels.
func (r domainPerListenerLabels) toMetricsLabels() []metrics.Label {
	return []metrics.Label{
		{Name: "namespace", Value: r.Namespace},
		{Name: "gatewayName", Value: r.GatewayName},
		{Name: "port", Value: r.Port},
	}
}

// SetDomainPerListener sets the number of domains per listener gauge.
func setDomainPerListener(labels domainPerListenerLabels, domains int) {
	domainsPerListener.Set(float64(domains), labels.toMetricsLabels()...)
}

// GetDomainsPerListener returns the domains per listener gauge.
// This is provided for testing purposes.
func GetDomainsPerListener() metrics.Gauge {
	return domainsPerListener
}
