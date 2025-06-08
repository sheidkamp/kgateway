package proxy_syncer

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/kgateway-dev/kgateway/v2/internal/kgateway/metrics"
	kmetrics "github.com/kgateway-dev/kgateway/v2/pkg/metrics"
	"github.com/kgateway-dev/kgateway/v2/pkg/metrics/metricstest"
)

func setupMetricsTest() {
	metrics.GetStatusSyncDuration().Reset()
	metrics.GetStatusSyncsTotal().Reset()
	metrics.GetStatusSyncResources().Reset()
}

func TestProxySyncerMetrics(t *testing.T) {
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
		xdsSnapshotsMetrics:   metrics.NewCollectionRecorder("ClientXDSSnapshots"),
	}

	testRouteMetrics(t, proxySyncer)
	testGatewayMetrics(t, proxySyncer)
	testListenerMetrics(t, proxySyncer)
	testPolicyMetrics(t, proxySyncer)
	testXDSSnapshotMetrics(t, proxySyncer)
}

func testRouteMetrics(t *testing.T, proxySyncer *ProxySyncer) {
	setupMetricsTest()

	finish := proxySyncer.routeStatusMetrics.StatusSyncStart()
	finish(nil)

	proxySyncer.routeStatusMetrics.SetResources(metrics.StatusSyncResourcesLabels{
		Namespace: "default",
		Name:      "test",
		Resource:  "HTTPRoute",
	}, 5)
	proxySyncer.routeStatusMetrics.SetResources(metrics.StatusSyncResourcesLabels{
		Namespace: "kube-system",
		Name:      "test",
		Resource:  "TCPRoute",
	}, 3)

	expectedSyncsCounter := `
		# HELP kgateway_status_syncer_status_syncs_total Total status syncs
		# TYPE kgateway_status_syncer_status_syncs_total counter
		kgateway_status_syncer_status_syncs_total{result="success",syncer="RouteStatusSyncer"} 1
	`

	err := metricstest.CollectAndCompare(
		metrics.GetStatusSyncsTotal(),
		strings.NewReader(expectedSyncsCounter),
		"kgateway_status_syncer_status_syncs_total",
	)
	require.NoError(t, err, "Route status translation counter mismatch")

	expectedResourceGauge := `
		# HELP kgateway_status_syncer_resources Current number of resources managed by the status syncer
		# TYPE kgateway_status_syncer_resources gauge
		kgateway_status_syncer_resources{name="test",namespace="default",resource="HTTPRoute",syncer="RouteStatusSyncer"} 5
		kgateway_status_syncer_resources{name="test",namespace="kube-system",resource="TCPRoute",syncer="RouteStatusSyncer"} 3
	`

	err = metricstest.CollectAndCompare(
		metrics.GetStatusSyncResources(),
		strings.NewReader(expectedResourceGauge),
		"kgateway_status_syncer_resources",
	)
	require.NoError(t, err, "Route status resource gauge mismatch")
}

func testGatewayMetrics(t *testing.T, proxySyncer *ProxySyncer) {
	setupMetricsTest()

	finish := proxySyncer.gatewayStatusMetrics.StatusSyncStart()
	finish(errors.New("gateway sync error"))

	proxySyncer.gatewayStatusMetrics.SetResources(metrics.StatusSyncResourcesLabels{
		Namespace: "default",
		Name:      "test",
		Resource:  "Gateway",
	}, 2)

	gathered := metricstest.MustGatherMetrics(t)

	gathered.AssertMetricExists("kgateway_status_syncer_status_syncs_total")
	gathered.AssertMetricsLabels(
		"kgateway_status_syncer_status_syncs_total",
		[][]kmetrics.Label{{
			{Name: "result", Value: "error"},
			{Name: "syncer", Value: "GatewayStatusSyncer"},
		}},
	)
}

func testListenerMetrics(t *testing.T, proxySyncer *ProxySyncer) {
	setupMetricsTest()

	proxySyncer.listenerStatusMetrics.IncResources(metrics.StatusSyncResourcesLabels{
		Namespace: "default",
		Name:      "test",
		Resource:  "ListenerSet",
	})
	proxySyncer.listenerStatusMetrics.IncResources(metrics.StatusSyncResourcesLabels{
		Namespace: "default",
		Name:      "test",
		Resource:  "ListenerSet",
	})
	proxySyncer.listenerStatusMetrics.DecResources(metrics.StatusSyncResourcesLabels{
		Namespace: "default",
		Name:      "test",
		Resource:  "ListenerSet",
	})

	gathered := metricstest.MustGatherMetrics(t)

	gathered.AssertMetricExists("kgateway_status_syncer_resources")
	gathered.AssertMetricsLabels(
		"kgateway_status_syncer_resources",
		[][]kmetrics.Label{{
			{Name: "name", Value: "test"},
			{Name: "namespace", Value: "default"},
			{Name: "resource", Value: "ListenerSet"},
			{Name: "syncer", Value: "ListenerSetStatusSyncer"},
		}},
	)
}

func testPolicyMetrics(t *testing.T, proxySyncer *ProxySyncer) {
	setupMetricsTest()

	finish := proxySyncer.policyStatusMetrics.StatusSyncStart()
	finish(nil)

	proxySyncer.policyStatusMetrics.SetResources(metrics.StatusSyncResourcesLabels{
		Namespace: "default",
		Name:      "test",
		Resource:  "Policy",
	}, 7)

	gathered := metricstest.MustGatherMetrics(t)

	gathered.AssertMetricExists("kgateway_status_syncer_status_syncs_total")
	gathered.AssertMetricsLabels(
		"kgateway_status_syncer_status_syncs_total",
		[][]kmetrics.Label{{
			{Name: "result", Value: "success"},
			{Name: "syncer", Value: "PolicyStatusSyncer"},
		}},
	)

	gathered.AssertMetricExists("kgateway_status_syncer_resources")
	gathered.AssertMetricsLabels(
		"kgateway_status_syncer_resources",
		[][]kmetrics.Label{{
			{Name: "name", Value: "test"},
			{Name: "namespace", Value: "default"},
			{Name: "resource", Value: "Policy"},
			{Name: "syncer", Value: "PolicyStatusSyncer"},
		}},
	)
}

func testXDSSnapshotMetrics(t *testing.T, proxySyncer *ProxySyncer) {
	setupMetricsTest()

	finish := proxySyncer.xdsSnapshotsMetrics.TransformStart()
	finish(nil)

	proxySyncer.xdsSnapshotsMetrics.SetResources(metrics.CollectionResourcesLabels{
		Namespace: "default",
		Name:      "test",
		Resource:  "Cluster",
	}, 7)

	gathered := metricstest.MustGatherMetrics(t)

	gathered.AssertMetricExists("kgateway_collection_transforms_total")
	gathered.AssertMetricsLabels(
		"kgateway_collection_transforms_total",
		[][]kmetrics.Label{{
			{Name: "collection", Value: "ClientXDSSnapshots"},
			{Name: "result", Value: "success"},
		}},
	)

	gathered.AssertMetricExists("kgateway_collection_resources")
	gathered.AssertMetricsLabels(
		"kgateway_collection_resources",
		[][]kmetrics.Label{{
			{Name: "collection", Value: "ClientXDSSnapshots"},
			{Name: "name", Value: "test"},
			{Name: "namespace", Value: "default"},
			{Name: "resource", Value: "Cluster"},
		}},
	)
}

func testMetricsLinting(t *testing.T) {
	setupMetricsTest()

	routeRecorder := metrics.NewStatusSyncRecorder("TestSyncer")

	finish := routeRecorder.StatusSyncStart()
	finish(nil)

	routeRecorder.SetResources(metrics.StatusSyncResourcesLabels{
		Namespace: "default",
		Name:      "test",
		Resource:  "HTTPRoute",
	}, 1)

	xdsSnapshotRecorder := metrics.NewCollectionRecorder("TestXDSSnapshots")

	finish = xdsSnapshotRecorder.TransformStart()
	finish(nil)

	xdsSnapshotRecorder.SetResources(metrics.CollectionResourcesLabels{
		Namespace: "default",
		Name:      "test",
		Resource:  "Cluster",
	}, 2)

	problems, err := metricstest.GatherAndLint()
	require.NoError(t, err)

	if len(problems) > 0 {
		t.Errorf("Metrics linting problems found: %v", problems)
	}

	gathered := metricstest.MustGatherMetrics(t)

	expectedMetrics := []string{
		"kgateway_status_syncer_status_syncs_total",
		"kgateway_status_syncer_status_sync_duration_seconds",
		"kgateway_status_syncer_resources",
		"kgateway_collection_transforms_total",
		"kgateway_collection_transform_duration_seconds",
		"kgateway_collection_resources",
	}

	for _, expected := range expectedMetrics {
		gathered.AssertMetricExists(expected)
	}
}
