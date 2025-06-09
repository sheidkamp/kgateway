package proxy_syncer

import (
	"fmt"
	"strings"

	envoycachetypes "github.com/envoyproxy/go-control-plane/pkg/cache/types"
	envoycache "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"istio.io/istio/pkg/kube/controllers"
	"istio.io/istio/pkg/kube/krt"

	"github.com/kgateway-dev/kgateway/v2/internal/kgateway/ir"
	"github.com/kgateway-dev/kgateway/v2/internal/kgateway/krtcollections"
	"github.com/kgateway-dev/kgateway/v2/internal/kgateway/utils/krtutil"
)

func snapshotPerClient(
	krtopts krtutil.KrtOptions,
	uccCol krt.Collection[ir.UniqlyConnectedClient],
	mostXdsSnapshots krt.Collection[GatewayXdsResources],
	endpoints PerClientEnvoyEndpoints,
	clusters PerClientEnvoyClusters,
	metricsRecorder krtcollections.CollectionMetricsRecorder,
) krt.Collection[XdsSnapWrapper] {
	xdsSnapshotsForUcc := krt.NewCollection(uccCol, func(kctx krt.HandlerContext, ucc ir.UniqlyConnectedClient) *XdsSnapWrapper {
		defer metricsRecorder.TransformStart()(nil)

		maybeMostlySnap := krt.FetchOne(kctx, mostXdsSnapshots, krt.FilterKey(ucc.Role))
		if maybeMostlySnap == nil {
			logger.Debug("snapshot missing", "proxy_key", ucc.Role)
			return nil
		}
		clustersForUcc := clusters.FetchClustersForClient(kctx, ucc)

		logger.Debug("found perclient clusters", "client", ucc.ResourceName(), "clusters", len(clustersForUcc))

		// HACK
		// https://github.com/solo-io/gloo/pull/10611/files#diff-060acb7cdd3a287a3aef1dd864aae3e0193da17b6230c382b649ce9dc0eca80b
		// Without this, we will send a "blip" where the DestinationRule
		// or other per-client config is not applied to the clusters
		// by sending the genericSnap clusters on the first pass, then
		// the correct ones.
		// This happens because the event for the new connected client
		// triggers the per-client cluster transformation in parallel
		// with this snapshotPerClient transformation. This Fetch is racing
		// with that computation and will almost always lose.
		// While we're looking for a way to make this ordering predictable
		// to avoid hacks like this, it will do for now.
		if len(clustersForUcc) == 0 {
			logger.Info("no perclient clusters; defer building snapshot", "client", ucc.ResourceName())
			return nil
		}

		clustersProto := make([]envoycachetypes.ResourceWithTTL, 0, len(clustersForUcc)+len(maybeMostlySnap.Clusters))
		var clustersHash uint64
		var erroredClusters []string
		for _, c := range clustersForUcc {
			if c.Error == nil {
				clustersProto = append(clustersProto, envoycachetypes.ResourceWithTTL{Resource: c.Cluster})
				clustersHash ^= c.ClusterVersion
			} else {
				erroredClusters = append(erroredClusters, c.Name)
			}
		}
		clustersProto = append(clustersProto, maybeMostlySnap.Clusters...)
		clustersHash ^= maybeMostlySnap.ClustersHash
		clustersVersion := fmt.Sprintf("%d", clustersHash)

		endpointsForUcc := endpoints.FetchEndpointsForClient(kctx, ucc)
		endpointsProto := make([]envoycachetypes.ResourceWithTTL, 0, len(endpointsForUcc))
		var endpointsHash uint64
		for _, ep := range endpointsForUcc {
			endpointsProto = append(endpointsProto, envoycachetypes.ResourceWithTTL{Resource: ep.Endpoints})
			endpointsHash ^= ep.EndpointsHash
		}

		snap := XdsSnapWrapper{}

		clusterResources := envoycache.NewResourcesWithTTL(clustersVersion, clustersProto)
		endpointResources := envoycache.NewResourcesWithTTL(fmt.Sprintf("%d", endpointsHash), endpointsProto)
		snap.erroredClusters = erroredClusters
		snap.proxyKey = ucc.ResourceName()
		snapshot := &envoycache.Snapshot{}
		snapshot.Resources[envoycachetypes.Cluster] = clusterResources // envoycache.NewResources(version, resource)
		snapshot.Resources[envoycachetypes.Endpoint] = endpointResources
		snapshot.Resources[envoycachetypes.Route] = maybeMostlySnap.Routes
		snapshot.Resources[envoycachetypes.Listener] = maybeMostlySnap.Listeners
		// envoycache.NewResources(version, resource)
		snap.snap = snapshot
		logger.Debug("snapshots", "proxy_key", snap.proxyKey,
			"listeners", resourcesStringer(maybeMostlySnap.Listeners).String(),
			"clusters", resourcesStringer(clusterResources).String(),
			"routes", resourcesStringer(maybeMostlySnap.Routes).String(),
			"endpoints", resourcesStringer(endpointResources).String(),
		)

		return &snap
	}, krtopts.ToOptions("PerClientXdsSnapshots")...)

	xdsSnapshotsForUcc.Register(func(o krt.Event[XdsSnapWrapper]) {
		name := o.Latest().ResourceName()
		namespace := "unknown"

		pks := strings.SplitN(name, "~", 5)
		if len(pks) > 1 {
			namespace = pks[1]
		}

		if len(pks) > 2 {
			name = pks[2]
		}

		switch o.Event {
		case controllers.EventDelete:
			metricsRecorder.SetResources(krtcollections.CollectionResourcesMetricLabels{
				Namespace: namespace,
				Name:      name,
				Resource:  "Cluster",
			}, 0)

			metricsRecorder.SetResources(krtcollections.CollectionResourcesMetricLabels{
				Namespace: namespace,
				Name:      name,
				Resource:  "Endpoint",
			}, 0)

			metricsRecorder.SetResources(krtcollections.CollectionResourcesMetricLabels{
				Namespace: namespace,
				Name:      name,
				Resource:  "Route",
			}, 0)

			metricsRecorder.SetResources(krtcollections.CollectionResourcesMetricLabels{
				Namespace: namespace,
				Name:      name,
				Resource:  "Listener",
			}, 0)
		case controllers.EventAdd, controllers.EventUpdate:
			metricsRecorder.SetResources(krtcollections.CollectionResourcesMetricLabels{
				Namespace: namespace,
				Name:      name,
				Resource:  "Cluster",
			}, len(o.Latest().snap.Resources[envoycachetypes.Cluster].Items))

			metricsRecorder.SetResources(krtcollections.CollectionResourcesMetricLabels{
				Namespace: namespace,
				Name:      name,
				Resource:  "Endpoint",
			}, len(o.Latest().snap.Resources[envoycachetypes.Endpoint].Items))

			metricsRecorder.SetResources(krtcollections.CollectionResourcesMetricLabels{
				Namespace: namespace,
				Name:      name,
				Resource:  "Route",
			}, len(o.Latest().snap.Resources[envoycachetypes.Route].Items))

			metricsRecorder.SetResources(krtcollections.CollectionResourcesMetricLabels{
				Namespace: namespace,
				Name:      name,
				Resource:  "Listener",
			}, len(o.Latest().snap.Resources[envoycachetypes.Listener].Items))
		}
	})

	return xdsSnapshotsForUcc
}
