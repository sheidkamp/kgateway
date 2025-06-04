package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

const (
	routeSubsystem = "route"
)

var (
	domainsPerListener = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Subsystem: routeSubsystem,
			Name:      "domains",
			Help:      "Number of domains per listener",
		},
		[]string{"namespace", "gatewayName", "port"},
	)
)

// DomainPerListenerLabels is used as an argument to SetDomainPerListener
type DomainPerListenerLabels struct {
	Namespace   string
	GatewayName string
	Port        string
}

func (r DomainPerListenerLabels) toMetricsLabels() []string {
	return []string{r.Namespace, r.GatewayName, r.Port}
}

type RouteRecorder interface {
	SetDomainPerListener(labels DomainPerListenerLabels, domains int)
}

var _ RouteRecorder = &routeMetrics{}

// TranslatorMetrics provides metrics for translator operations.
type routeMetrics struct {
	domainsPerListener *prometheus.GaugeVec
}

// NewRouteRecorder creates a new RouteRecorder
func NewRouteRecorder() RouteRecorder {
	return &routeMetrics{
		domainsPerListener: domainsPerListener,
	}
}

func (m *routeMetrics) SetDomainPerListener(labels DomainPerListenerLabels, domains int) {
	m.domainsPerListener.WithLabelValues(labels.toMetricsLabels()...).Set(float64(domains))
}
