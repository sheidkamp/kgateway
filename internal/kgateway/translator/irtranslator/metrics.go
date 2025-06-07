package irtranslator

import (
	"github.com/kgateway-dev/kgateway/v2/internal/kgateway/metrics"
)

var (
	m                  = metrics.NewMetrics()
	domainsPerListener = m.NewGaugeVec(
		metrics.GaugeOpts{
			Namespace: metrics.MetricsNamespaceKGateway,
			Subsystem: metrics.MetricsSubsystemRouting,
			Name:      "domains",
			Help:      "Number of domains per listener",
		},
		[]string{"namespace", "gatewayName", "port"},
	)
)

func init() {
	m.RegisterMetrics([]metrics.Collector{
		domainsPerListener,
	})
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

func setDomainPerListener(labels DomainPerListenerLabels, domains int) {
	domainsPerListener.WithLabelValues(labels.toMetricsLabels()...).Set(float64(domains))
}
