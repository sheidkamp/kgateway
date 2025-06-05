package setup

import (
	"context"

	xdsserver "github.com/envoyproxy/go-control-plane/pkg/server/v3"
	core "github.com/kgateway-dev/kgateway/v2/internal/kgateway/setup"
	sdk "github.com/kgateway-dev/kgateway/v2/pkg/pluginsdk"
	common "github.com/kgateway-dev/kgateway/v2/pkg/pluginsdk/collections"
)

type Options struct {
	ExtraPlugins      func(ctx context.Context, commoncol *common.CommonCollections) []sdk.Plugin
	ExtraXDSCallbacks xdsserver.Callbacks
}

func New(opts Options) core.Server {
	// internal setup already accepted functional-options; we wrap only extras.
	return core.New(core.WithExtraPlugins(opts.ExtraPlugins), core.WithExtraXDSCallbacks(opts.ExtraXDSCallbacks))
}
