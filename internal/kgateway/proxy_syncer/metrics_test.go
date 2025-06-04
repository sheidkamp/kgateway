package proxy_syncer

import (
	"errors"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	crmetrics "sigs.k8s.io/controller-runtime/pkg/metrics"

	"github.com/kgateway-dev/kgateway/v2/internal/kgateway/metrics"
)

func setupMetricsTest() {
	metrics.ResetStatusSyncMetrics()
}

func TestProxySyncerMetrics(t *testing.T) {
	setupMetricsTest()

	t.Run("SyncMetrics", func(t *testing.T) {
		testProxySyncerMetrics(t)
	})

	t.Run("MetricsLinting", func(t *testing.T) {
		testMetricsLinting(t)
	})
}

func testProxySyncerMetrics(t *testing.T) {
	proxySyncer := &ProxySyncer{
		routeStatusMetrics:    metrics.NewStatusSyncRecorder("RouteStatusSyncer"),
		gatewayStatusMetrics:  metrics.NewStatusSyncRecorder("GatewayStatusSyncer"),
		listenerStatusMetrics: metrics.NewStatusSyncRecorder("ListenerSetStatusSyncer"),
		policyStatusMetrics:   metrics.NewStatusSyncRecorder("PolicyStatusSyncer"),
	}

	testRouteMetrics(t, proxySyncer)
	testGatewayMetrics(t, proxySyncer)
	testListenerMetrics(t, proxySyncer)
	testPolicyMetrics(t, proxySyncer)
}

func testRouteMetrics(t *testing.T, proxySyncer *ProxySyncer) {
	finish := proxySyncer.routeStatusMetrics.StatusSyncStart()
	finish(nil)

	proxySyncer.routeStatusMetrics.SetResources("default", "test", "HTTPRoute", 5)
	proxySyncer.routeStatusMetrics.SetResources("kube-system", "test", "TCPRoute", 3)

	expectedTranslationCounter := `
		# HELP kgateway_status_syncer_status_syncs_total Total translations
		# TYPE kgateway_status_syncer_status_syncs_total counter
		kgateway_status_syncer_status_syncs_total{result="success",syncer="RouteStatusSyncer"} 1
	`

	err := testutil.CollectAndCompare(
		metrics.GetStatusSyncsTotal(),
		strings.NewReader(expectedTranslationCounter),
		"kgateway_kgateway_status_syncer_status_syncs_total",
	)
	require.NoError(t, err, "Route status translation counter mismatch")

	expectedResourceGauge := `
		# HELP kgateway_status_syncer_resources Current number of resources managed by the status syncer
		# TYPE kgateway_status_syncer_resources gauge
		kgateway_status_syncer_resources{name="test",namespace="default",resource="HTTPRoute",syncer="RouteStatusSyncer"} 5
		kgateway_status_syncer_resources{name="test",namespace="kube-system",resource="TCPRoute",syncer="RouteStatusSyncer"} 3
	`

	err = testutil.CollectAndCompare(
		metrics.GetStatusSyncResources(),
		strings.NewReader(expectedResourceGauge),
		"kgateway_status_syncer_resources",
	)
	require.NoError(t, err, "Route status resource gauge mismatch")
}

func testGatewayMetrics(t *testing.T, proxySyncer *ProxySyncer) {
	finish := proxySyncer.gatewayStatusMetrics.StatusSyncStart()
	finish(errors.New("gateway sync error"))

	proxySyncer.gatewayStatusMetrics.SetResources("default", "test", "Gateway", 2)

	metricFamilies, err := crmetrics.Registry.Gather()
	require.NoError(t, err)

	var found bool

	for _, mf := range metricFamilies {
		if *mf.Name == "kgateway_status_syncer_status_syncs_total" {
			for _, metric := range mf.Metric {
				if len(metric.Label) >= 2 &&
					*metric.Label[0].Value == "error" &&
					*metric.Label[1].Value == "GatewayStatusSyncer" {
					found = true
					assert.Equal(t, float64(1), metric.Counter.GetValue())
				}
			}
		}
	}

	assert.True(t, found, "Gateway status error metric not found")
}

func testListenerMetrics(t *testing.T, proxySyncer *ProxySyncer) {
	proxySyncer.listenerStatusMetrics.IncResources("default", "test", "ListenerSet")
	proxySyncer.listenerStatusMetrics.IncResources("default", "test", "ListenerSet")
	proxySyncer.listenerStatusMetrics.DecResources("default", "test", "ListenerSet")

	metricFamilies, err := crmetrics.Registry.Gather()
	require.NoError(t, err)

	var found bool

	for _, mf := range metricFamilies {
		if *mf.Name == "kgateway_status_syncer_resources" {
			for _, metric := range mf.Metric {
				if len(metric.Label) == 4 &&
					*metric.Label[0].Value == "test" &&
					*metric.Label[1].Value == "default" &&
					*metric.Label[2].Value == "ListenerSet" &&
					*metric.Label[3].Value == "ListenerSetStatusSyncer" {
					found = true
					assert.Equal(t, float64(1), metric.Gauge.GetValue())
				}
			}
		}
	}

	assert.True(t, found, "Listener status resource metric not found")
}

func testPolicyMetrics(t *testing.T, proxySyncer *ProxySyncer) {
	finish := proxySyncer.policyStatusMetrics.StatusSyncStart()
	finish(nil)

	proxySyncer.policyStatusMetrics.SetResources("default", "test", "Policy", 7)

	metricFamilies, err := crmetrics.Registry.Gather()
	require.NoError(t, err)

	foundTranslation := false
	foundResource := false

	for _, mf := range metricFamilies {
		if *mf.Name == "kgateway_status_syncer_status_syncs_total" {
			for _, metric := range mf.Metric {
				if len(metric.Label) >= 2 &&
					*metric.Label[0].Value == "success" &&
					*metric.Label[1].Value == "PolicyStatusSyncer" {
					foundTranslation = true
					assert.Equal(t, float64(1), metric.Counter.GetValue())
				}
			}
		}

		if *mf.Name == "kgateway_status_syncer_resources" {
			for _, metric := range mf.Metric {
				if len(metric.Label) == 4 &&
					*metric.Label[0].Value == "test" &&
					*metric.Label[1].Value == "default" &&
					*metric.Label[2].Value == "Policy" &&
					*metric.Label[3].Value == "PolicyStatusSyncer" {
					foundResource = true
					assert.Equal(t, float64(7), metric.Gauge.GetValue())
				}
			}
		}
	}

	assert.True(t, foundTranslation, "Policy status translation metric not found")
	assert.True(t, foundResource, "Policy status resource metric not found")
}

func testMetricsLinting(t *testing.T) {
	routeRecorder := metrics.NewStatusSyncRecorder("TestSyncer")

	finish := routeRecorder.StatusSyncStart()
	finish(nil)

	routeRecorder.SetResources("default", "test", "HTTPRoute", 1)

	reg := prometheus.NewRegistry()
	reg.MustRegister(
		metrics.GetStatusSyncsTotal(),
		metrics.GetStatusSyncDuration(),
		metrics.GetStatusSyncResources(),
	)

	problems, err := testutil.GatherAndLint(reg)
	require.NoError(t, err)

	if len(problems) > 0 {
		t.Errorf("Metrics linting problems found: %v", problems)
	}

	metricFamilies, err := crmetrics.Registry.Gather()
	require.NoError(t, err)

	metricNames := make(map[string]bool)
	for _, mf := range metricFamilies {
		metricNames[*mf.Name] = true
	}

	expectedMetrics := []string{
		"kgateway_status_syncer_status_syncs_total",
		"kgateway_status_syncer_status_sync_duration_seconds",
		"kgateway_status_syncer_resources",
	}

	for _, expected := range expectedMetrics {
		assert.True(t, metricNames[expected], "Expected metric %s not found", expected)
	}
}
