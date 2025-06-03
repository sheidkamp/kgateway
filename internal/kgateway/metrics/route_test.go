package metrics_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	. "github.com/kgateway-dev/kgateway/v2/internal/kgateway/metrics"
)

var (
	route = &gwv1.HTTPRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-route",
			Namespace: "test-namespace",
		},
	}
)

func TestNewRouteMetrics(t *testing.T) {
	setupTest()

	m := NewRouteRecorder()

	m.SetListenerCount(route, "test-type", 5)

	metricFamilies, err := metrics.Registry.Gather()
	require.NoError(t, err)

	expectedMetrics := []string{
		"kgateway_route_listener_counts",
	}

	foundMetrics := make(map[string]bool)
	for _, mf := range metricFamilies {
		foundMetrics[*mf.Name] = true
	}

	for _, expected := range expectedMetrics {
		assert.True(t, foundMetrics[expected], "Expected metric %s not found", expected)
	}
}

func TestSetListenerCount(t *testing.T) {
	setupTest()

	m := NewRouteRecorder()

	m.SetListenerCount(route, "test-type", 5)

	metricFamilies, err := metrics.Registry.Gather()
	require.NoError(t, err)

	var found bool
	for _, mf := range metricFamilies {
		if *mf.Name == "kgateway_route_listener_counts" {
			found = true
			assert.Equal(t, 1, len(mf.Metric))
			assert.Equal(t, "test-route", *mf.Metric[0].Label[0].Value)
			assert.Equal(t, "test-namespace", *mf.Metric[0].Label[1].Value)
			assert.Equal(t, "test-type", *mf.Metric[0].Label[2].Value)
			assert.Equal(t, float64(5), mf.Metric[0].Gauge.GetValue())
		}
	}

	assert.True(t, found, "kgateway_route_listener_counts metric not found")
}
