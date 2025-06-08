package metrics_test

import (
	"testing"

	. "github.com/kgateway-dev/kgateway/v2/internal/kgateway/metrics"
	"github.com/kgateway-dev/kgateway/v2/pkg/metrics"
	"github.com/kgateway-dev/kgateway/v2/pkg/metrics/metricstest"
)

var (
	namespace = "test-namespace"
	gateway   = "test-gateway"
	port      = "80"
)

func TestNewRoutingRecorder(t *testing.T) {
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

	currentMetrics := metricstest.MustGatherMetrics(t)

	for _, expected := range expectedMetrics {
		currentMetrics.AssertMetricExists(expected)
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

	currentMetrics := metricstest.MustGatherMetrics(t)

	currentMetrics.AssertMetricLabels("kgateway_routing_domains", []metrics.Label{
		{Name: "gatewayName", Value: gateway},
		{Name: "namespace", Value: namespace},
		{Name: "port", Value: port},
	})
	currentMetrics.AssertMetricGaugeValue("kgateway_routing_domains", 5)
}
