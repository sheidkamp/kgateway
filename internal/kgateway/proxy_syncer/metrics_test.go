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
	metrics.ResetSnapshotMetrics()
}

func TestProxySyncerMetrics(t *testing.T) {
	setupMetricsTest()

	t.Run("SyncMetrics", func(t *testing.T) {
		testProxySyncerMetrics(t)
	})

	t.Run("SnapshotSyncMetrics", func(t *testing.T) {
		testProxyTranslatorSnapshotMetrics(t)
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

	proxySyncer.routeStatusMetrics.SetResources("default", "HTTPRoute", 5)
	proxySyncer.routeStatusMetrics.SetResources("kube-system", "TCPRoute", 3)

	expectedTranslationCounter := `
		# HELP kgateway_translator_translations_total Total translations
		# TYPE kgateway_translator_translations_total counter
		kgateway_translator_translations_total{result="success",translator="RouteStatusSyncer"} 1
	`

	err := testutil.CollectAndCompare(
		metrics.GetStatusSyncsTotal(),
		strings.NewReader(expectedTranslationCounter),
		"kgateway_translator_translations_total",
	)
	require.NoError(t, err, "Route status translation counter mismatch")

	expectedResourceGauge := `
		# HELP kgateway_translator_resources Current number of resources managed by the translator
		# TYPE kgateway_translator_resources gauge
		kgateway_translator_resources{namespace="default",resource="HTTPRoute",translator="RouteStatusSyncer"} 5
		kgateway_translator_resources{namespace="kube-system",resource="TCPRoute",translator="RouteStatusSyncer"} 3
	`

	err = testutil.CollectAndCompare(
		metrics.GetStatusSyncResources(),
		strings.NewReader(expectedResourceGauge),
		"kgateway_translator_resources",
	)
	require.NoError(t, err, "Route status resource gauge mismatch")
}

func testGatewayMetrics(t *testing.T, proxySyncer *ProxySyncer) {
	finish := proxySyncer.gatewayStatusMetrics.StatusSyncStart()
	finish(errors.New("gateway sync error"))

	proxySyncer.gatewayStatusMetrics.SetResources("default", "Gateway", 2)

	metricFamilies, err := crmetrics.Registry.Gather()
	require.NoError(t, err)

	var found bool

	for _, mf := range metricFamilies {
		if *mf.Name == "kgateway_translator_translations_total" {
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
	proxySyncer.listenerStatusMetrics.IncResources("default", "ListenerSet")
	proxySyncer.listenerStatusMetrics.IncResources("default", "ListenerSet")
	proxySyncer.listenerStatusMetrics.DecResources("default", "ListenerSet")

	metricFamilies, err := crmetrics.Registry.Gather()
	require.NoError(t, err)

	var found bool

	for _, mf := range metricFamilies {
		if *mf.Name == "kgateway_translator_resources" {
			for _, metric := range mf.Metric {
				if len(metric.Label) >= 3 &&
					*metric.Label[0].Value == "default" &&
					*metric.Label[1].Value == "ListenerSet" &&
					*metric.Label[2].Value == "ListenerSetStatusSyncer" {
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

	proxySyncer.policyStatusMetrics.SetResources("default", "Policy", 7)

	metricFamilies, err := crmetrics.Registry.Gather()
	require.NoError(t, err)

	foundTranslation := false
	foundResource := false

	for _, mf := range metricFamilies {
		if *mf.Name == "kgateway_translator_translations_total" {
			for _, metric := range mf.Metric {
				if len(metric.Label) >= 2 &&
					*metric.Label[0].Value == "success" &&
					*metric.Label[1].Value == "PolicyStatusSyncer" {
					foundTranslation = true
					assert.Equal(t, float64(1), metric.Counter.GetValue())
				}
			}
		}

		if *mf.Name == "kgateway_translator_resources" {
			for _, metric := range mf.Metric {
				if len(metric.Label) >= 3 &&
					*metric.Label[0].Value == "default" &&
					*metric.Label[1].Value == "Policy" &&
					*metric.Label[2].Value == "PolicyStatusSyncer" {
					foundResource = true
					assert.Equal(t, float64(7), metric.Gauge.GetValue())
				}
			}
		}
	}

	assert.True(t, foundTranslation, "Policy status translation metric not found")
	assert.True(t, foundResource, "Policy status resource metric not found")
}

func testProxyTranslatorSnapshotMetrics(t *testing.T) {
	proxyTranslator := NewProxyTranslator(nil)
	proxyKey := "test-proxy"

	finish := proxyTranslator.metrics.SnapshotStart(proxyKey)
	finish(nil)

	proxyTranslator.metrics.SetResources(proxyKey, 42)

	expectedSnapshotCounter := `
		# HELP kgateway_snapshot_syncs_total Total snapshot syncs
		# TYPE kgateway_snapshot_syncs_total counter
		kgateway_snapshot_syncs_total{proxy="test-proxy",result="success",snapshot="ProxyTranslator"} 1
	`

	err := testutil.CollectAndCompare(
		metrics.GetSnapshotSyncsTotal(),
		strings.NewReader(expectedSnapshotCounter),
		"kgateway_snapshot_syncs_total",
	)
	require.NoError(t, err, "Snapshot sync counter mismatch")

	expectedSnapshotGauge := `
		# HELP kgateway_snapshot_resources Current number of resources contained in the snapshot
		# TYPE kgateway_snapshot_resources gauge
		kgateway_snapshot_resources{proxy="test-proxy",snapshot="ProxyTranslator"} 42
	`

	err = testutil.CollectAndCompare(
		metrics.GetSnapshotResources(),
		strings.NewReader(expectedSnapshotGauge),
		"kgateway_snapshot_resources",
	)
	require.NoError(t, err, "Snapshot resource gauge mismatch")

	finishFuncError := proxyTranslator.metrics.SnapshotStart(proxyKey)
	finishFuncError(errors.New("snapshot error"))

	metricFamilies, err := crmetrics.Registry.Gather()
	require.NoError(t, err)

	var found bool
	for _, mf := range metricFamilies {
		if *mf.Name == "kgateway_snapshot_syncs_total" {
			for _, metric := range mf.Metric {
				if len(metric.Label) >= 3 &&
					*metric.Label[0].Value == proxyKey &&
					*metric.Label[1].Value == "error" &&
					*metric.Label[2].Value == "ProxyTranslator" {
					found = true
					assert.Equal(t, float64(1), metric.Counter.GetValue())
				}
			}
		}
	}
	assert.True(t, found, "Snapshot error metric not found")
}

func testMetricsLinting(t *testing.T) {
	routeRecorder := metrics.NewStatusSyncRecorder("TestTranslator")
	snapshotRecorder := metrics.NewSnapshotRecorder("TestSnapshot")

	finish := routeRecorder.StatusSyncStart()
	finish(nil)

	routeRecorder.SetResources("default", "HTTPRoute", 1)

	finishSnap := snapshotRecorder.SnapshotStart("test-proxy")
	finishSnap(nil)

	snapshotRecorder.SetResources("test-proxy", 10)

	reg := prometheus.NewRegistry()
	reg.MustRegister(
		metrics.GetStatusSyncsTotal(),
		metrics.GetStatusSyncDuration(),
		metrics.GetStatusSyncResources(),
		metrics.GetSnapshotSyncsTotal(),
		metrics.GetSnapshotSyncDuration(),
		metrics.GetSnapshotResources(),
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
		"kgateway_translator_translations_total",
		"kgateway_translator_translation_duration_seconds",
		"kgateway_translator_resources",
		"kgateway_snapshot_syncs_total",
		"kgateway_snapshot_sync_duration_seconds",
		"kgateway_snapshot_resources",
	}

	for _, expected := range expectedMetrics {
		assert.True(t, metricNames[expected], "Expected metric %s not found", expected)
	}
}
