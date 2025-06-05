package metrics_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/metrics"

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

	metricFamilies, err := metrics.Registry.Gather()
	require.NoError(t, err)

	expectedMetrics := []string{
		"kgateway_routing_domains",
	}

	foundMetrics := make(map[string]bool)
	for _, mf := range metricFamilies {
		foundMetrics[*mf.Name] = true
	}

	for _, expected := range expectedMetrics {
		assert.True(t, foundMetrics[expected], "Expected metric %s not found", expected)
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

	metricFamilies, err := metrics.Registry.Gather()
	require.NoError(t, err)

	var found bool
	for _, mf := range metricFamilies {
		if *mf.Name == "kgateway_routing_domains" {
			found = true
			assert.Equal(t, 1, len(mf.Metric))
			assert.Equal(t, gateway, *mf.Metric[0].Label[0].Value)
			assert.Equal(t, namespace, *mf.Metric[0].Label[1].Value)
			assert.Equal(t, port, *mf.Metric[0].Label[2].Value)
			assert.Equal(t, float64(5), mf.Metric[0].Gauge.GetValue())
		}
	}

	assert.True(t, found, "kgateway_routing_domains metric not found")
}
