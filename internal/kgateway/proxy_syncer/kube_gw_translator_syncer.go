package proxy_syncer

import (
	"context"

	tmetrics "github.com/kgateway-dev/kgateway/v2/internal/kgateway/translator/metrics"
	"github.com/kgateway-dev/kgateway/v2/pkg/logging"
)

func (s *ProxyTranslator) syncXds(
	ctx context.Context,
	snapWrap XdsSnapWrapper,
) {
	snap := snapWrap.snap
	proxyKey := snapWrap.proxyKey

	// TODO: handle errored clusters by fetching them from the previous snapshot and using the old cluster

	// stringifying the snapshot may be an expensive operation, so we'd like to avoid building the large
	// string if we're not even going to log it anyway
	logger.Debug("syncing xds snapshot", "proxy_key", proxyKey)

	logger.Log(ctx, logging.LevelTrace, "syncing xds snapshot", "proxy_key", proxyKey)

	gateway, namespace := getGatewayFromXDSSnapshotResourceName(snapWrap.ResourceName())

	// if the snapshot is not consistent, make it so
	// TODO: me may need to copy this to not change krt cache.
	// TODO: this is also may not be needed now that envoy has
	// a default initial fetch timeout
	// snap.MakeConsistent()
	s.xdsCache.SetSnapshot(ctx, proxyKey, snap)

	tmetrics.IncXDSSnapshotSync(gateway, namespace)

	tmetrics.EndResourceSync(tmetrics.ResourceSyncDetails{
		Namespace:    namespace,
		Gateway:      gateway,
		ResourceType: "XDSSnapshot",
		ResourceName: gateway,
	}, true, resourcesXDSSyncsTotal, resourcesXDSyncDuration)
}
