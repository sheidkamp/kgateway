package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	domainsPerListener = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: string(MetricsNamespaceKGateway),
			Subsystem: string(MetricsSubsystemRouting),
			Name:      "domains",
			Help:      "Number of domains per listener",
		},
		[]string{"namespace", "gatewayName", "port"},
	)
)

type RoutingRecorder interface {
	SetDomainPerListener(labels DomainPerListenerLabels, domains int)
}

var _ RoutingRecorder = &routingMetrics{}

// TranslatorMetrics provides metrics for translator operations.
type routingMetrics struct {
	domainsPerListener *prometheus.GaugeVec
}

// DomainPerListenerLabels is used as an argument to SetDomainPerListener
type DomainPerListenerLabels struct {
	Namespace   string
	GatewayName string
	Port        string
}

func (r DomainPerListenerLabels) toMetricsLabels() []string {
	return []string{r.Namespace, r.GatewayName, r.Port}
}

// NewRouteRecorder creates a new RouteRecorder
func NewRoutingRecorder() RoutingRecorder {
	return &routingMetrics{
		domainsPerListener: domainsPerListener,
	}
}

func (m *routingMetrics) SetDomainPerListener(labels DomainPerListenerLabels, domains int) {
	m.domainsPerListener.WithLabelValues(labels.toMetricsLabels()...).Set(float64(domains))
}

// GetDomainsPerListener returns the domains per listener gauge.
// This is provided for testing purposes.
func GetDomainsPerListener() *prometheus.GaugeVec {
	return domainsPerListener
}
