package metrics_test

import (
	"testing"

	. "github.com/kgateway-dev/kgateway/v2/internal/kgateway/metrics"
)

var (
	namespace = "test-namespace"
	gateway   = "test-gateway"
	port      = "80"
)

func TestNewRouteMetrics(t *testing.T) {
	setupTest()

	m := NewRoutingRecorder()

	m.SetDomainPerListener(DomainPerListenerLabels{
		Namespace:   namespace,
		GatewayName: gateway,
		Port:        port,
	}, 5)

	expectedMetrics := []string{
		"kgateway_routing_domains",
	}

	currentMetrics := mustGatherMetrics(t)

	for _, expected := range expectedMetrics {
		currentMetrics.assertMetricExists(expected)
	}
}

func TestSetDomainPerListener(t *testing.T) {
	setupTest()

	m := NewRoutingRecorder()

	m.SetDomainPerListener(DomainPerListenerLabels{
		Namespace:   namespace,
		GatewayName: gateway,
		Port:        port,
	}, 5)

	currentMetrics := mustGatherMetrics(t)

	currentMetrics.assertMetricLabels("kgateway_routing_domains", []*metricLabel{
		{name: "gatewayName", value: gateway},
		{name: "namespace", value: namespace},
		{name: "port", value: port},
	})
	currentMetrics.assertMetricGaugeValue("kgateway_routing_domains", 5)
}
