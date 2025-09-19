package setup

import (
	"context"

	xdsserver "github.com/envoyproxy/go-control-plane/pkg/server/v3"
	"istio.io/istio/pkg/kube/kubetypes"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	core "github.com/kgateway-dev/kgateway/v2/internal/kgateway/setup"
	agwplugins "github.com/kgateway-dev/kgateway/v2/pkg/agentgateway/plugins"
	"github.com/kgateway-dev/kgateway/v2/pkg/deployer"
	sdk "github.com/kgateway-dev/kgateway/v2/pkg/pluginsdk"
	common "github.com/kgateway-dev/kgateway/v2/pkg/pluginsdk/collections"
	"github.com/kgateway-dev/kgateway/v2/pkg/validator"
)

type Options struct {
	GatewayControllerName      string
	AgentgatewayControllerName string
	GatewayClassName           string
	WaypointGatewayClassName   string
	AgentgatewayClassName      string
	AdditionalGatewayClasses   map[string]*deployer.GatewayClassInfo
	ExtraPlugins               func(ctx context.Context, commoncol *common.CommonCollections, mergeSettingsJSON string) []sdk.Plugin
	ExtraAgwPlugins            func(ctx context.Context, agw *agwplugins.AgwCollections) []agwplugins.AgwPlugin
	ExtraGatewayParameters     func(cli client.Client, inputs *deployer.Inputs) []deployer.ExtraGatewayParameters
	ExtraXDSCallbacks          xdsserver.Callbacks
	RestConfig                 *rest.Config
	CtrlMgrOptions             func(context.Context) *ctrl.Options
	// extra controller manager config, like registering additional controllers
	ExtraManagerConfig []func(ctx context.Context, mgr manager.Manager, objectFilter kubetypes.DynamicObjectFilter) error
	// Validator is the validator to use for the controller.
	Validator validator.Validator
}

func New(opts Options) (core.Server, error) {
	// internal setup already accepted functional-options; we wrap only extras.
	return core.New(
		core.WithExtraPlugins(opts.ExtraPlugins),
		core.WithExtraAgwPlugins(opts.ExtraAgwPlugins),
		core.ExtraGatewayParameters(opts.ExtraGatewayParameters),
		core.WithGatewayControllerName(opts.GatewayControllerName),
		core.WithAgwControllerName(opts.AgentgatewayControllerName),
		core.WithGatewayClassName(opts.GatewayClassName),
		core.WithWaypointClassName(opts.WaypointGatewayClassName),
		core.WithAgentgatewayClassName(opts.AgentgatewayClassName),
		core.WithAdditionalGatewayClasses(opts.AdditionalGatewayClasses),
		core.WithExtraXDSCallbacks(opts.ExtraXDSCallbacks),
		core.WithRestConfig(opts.RestConfig),
		core.WithControllerManagerOptions(opts.CtrlMgrOptions),
		core.WithExtraManagerConfig(opts.ExtraManagerConfig...),
		core.WithValidator(opts.Validator),
	)
}
