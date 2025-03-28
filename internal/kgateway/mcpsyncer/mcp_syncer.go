package mcpsyncer

//go:generate go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28
//go:generate protoc --proto_path=. --go_out=. --go_opt=paths=source_relative target.proto
//go:generate protoc --proto_path=. --go_out=. --go_opt=paths=source_relative rbac.proto

import (
	"context"
	"fmt"
	"maps"
	"slices"

	envoytypes "github.com/envoyproxy/go-control-plane/pkg/cache/types"
	envoycache "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/solo-io/go-utils/contextutils"
	"google.golang.org/protobuf/proto"
	"istio.io/istio/pkg/config/schema/kubeclient"
	"istio.io/istio/pkg/kube"
	"istio.io/istio/pkg/kube/controllers"
	"istio.io/istio/pkg/kube/kclient"
	"istio.io/istio/pkg/kube/krt"
	"istio.io/istio/pkg/kube/kubetypes"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kgateway-dev/kgateway/v2/api/v1alpha1"
	"github.com/kgateway-dev/kgateway/v2/internal/kgateway/extensions2/common"
	extensionsplug "github.com/kgateway-dev/kgateway/v2/internal/kgateway/extensions2/plugin"
	"github.com/kgateway-dev/kgateway/v2/internal/kgateway/ir"
	"github.com/kgateway-dev/kgateway/v2/internal/kgateway/query"
	"github.com/kgateway-dev/kgateway/v2/internal/kgateway/reports"
	gwtranslator "github.com/kgateway-dev/kgateway/v2/internal/kgateway/translator/gateway"
	"github.com/kgateway-dev/kgateway/v2/internal/kgateway/utils"
	"github.com/kgateway-dev/kgateway/v2/internal/kgateway/utils/krtutil"
	"github.com/kgateway-dev/kgateway/v2/internal/kgateway/xds"
	"github.com/kgateway-dev/kgateway/v2/pkg/client/clientset/versioned"
)

const (
	TargetTypeUrl   = "type.googleapis.com/mcp.kgateway.dev.target.v1alpha1.Target"
	TargetConfigUrl = "type.googleapis.com/mcp.kgateway.dev.rbac.v1alpha1.Config"
)

func registerTypes(restConfig *rest.Config) {
	cli, err := versioned.NewForConfig(restConfig)
	if err != nil {
		panic(err)
	}

	mcpAuthPolicies := v1alpha1.SchemeGroupVersion.WithResource("mcpauthpolicies")
	// schema.GroupVersionResource{Group: v1alpha1.GroupVersion.Group, Version: "v1alpha2", Resource: "tcproutes"}
	kubeclient.Register[*v1alpha1.MCPAuthPolicy](
		mcpAuthPolicies,
		v1alpha1.SchemeGroupVersion.WithKind("MCPAuthPolicy"),
		func(c kubeclient.ClientGetter, namespace string, o metav1.ListOptions) (runtime.Object, error) {
			return cli.GatewayV1alpha1().MCPAuthPolicies(namespace).List(context.Background(), o)
		},
		func(c kubeclient.ClientGetter, namespace string, o metav1.ListOptions) (watch.Interface, error) {
			return cli.GatewayV1alpha1().MCPAuthPolicies(namespace).Watch(context.Background(), o)
		},
	)

}

type McpSyncer struct {
	commonCols     *common.CommonCollections
	translator     *mcpTranslator
	controllerName string

	xDS         krt.Collection[mcpXdsResources]
	xdsCache    envoycache.SnapshotCache
	istioClient kube.Client

	waitForSync []cache.InformerSynced
}

func NewMcpSyncer(
	ctx context.Context,
	controllerName string,
	mgr manager.Manager,
	client kube.Client,
	commonCols *common.CommonCollections,
	xdsCache envoycache.SnapshotCache,
) *McpSyncer {
	registerTypes(mgr.GetConfig())
	return &McpSyncer{
		commonCols:     commonCols,
		translator:     newTranslator(ctx, commonCols),
		controllerName: controllerName,
		xdsCache:       xdsCache,
		// mgr:            mgr,
		istioClient: client,
	}
}

type mcpXdsResources struct {
	types.NamespacedName

	reports     reports.ReportMap
	McpServices envoycache.Resources
	McpRbac     envoycache.Resources
}

func (r mcpXdsResources) ResourceName() string {
	return xds.OwnerNamespaceNameID("mcp-kgateway-kube-gateway-api", r.Namespace, r.Name)
}

func (r mcpXdsResources) Equals(in mcpXdsResources) bool {
	return r.NamespacedName == in.NamespacedName && report{r.reports}.Equals(report{in.reports}) && r.McpServices.Version == in.McpServices.Version && r.McpRbac.Version == in.McpRbac.Version
}

type envoyResourceWithName struct {
	inner   envoytypes.ResourceWithName
	version uint64
}

func (r envoyResourceWithName) ResourceName() string {
	return r.inner.GetName()
}

func (r envoyResourceWithName) Equals(in envoyResourceWithName) bool {
	return r.version == in.version
}

type envoyResourceWithCustomName struct {
	proto.Message
	Name    string
	version uint64
}

func (r envoyResourceWithCustomName) ResourceName() string {
	return r.Name
}

func (r envoyResourceWithCustomName) GetName() string {
	return r.Name
}

func (r envoyResourceWithCustomName) Equals(in envoyResourceWithCustomName) bool {
	return r.version == in.version
}

var _ envoytypes.ResourceWithName = envoyResourceWithCustomName{}

type rbacConfig struct {
	TargetRefs []types.NamespacedName
	// config     *Config
	Config envoyResourceWithCustomName
}

func (r rbacConfig) ResourceName() string {
	return r.Config.ResourceName()
}
func (r rbacConfig) Equals(in rbacConfig) bool {
	return slices.Equal(r.TargetRefs, in.TargetRefs) && r.Config.Equals(in.Config)
}

type mcpService struct {
	krt.Named
	ip   string
	port int
	path string
}

func (r mcpService) Equals(in mcpService) bool {
	return r.ip == in.ip && r.port == in.port && r.path == in.path
}

func (s *McpSyncer) Init(ctx context.Context, krtopts krtutil.KrtOptions) error {
	// find mcp gateways
	mcpauthpolicies := krt.WrapClient(kclient.NewDelayedInformer[*v1alpha1.MCPAuthPolicy](s.istioClient, v1alpha1.SchemeGroupVersion.WithResource("mcpauthpolicies"), kubetypes.StandardInformer, kclient.Filter{}), krtopts.ToOptions("MCPAuthPolicy")...)

	// convert to rbac json
	mcpRbac := krt.NewCollection(mcpauthpolicies, func(kctx krt.HandlerContext, mcpauthpolicy *v1alpha1.MCPAuthPolicy) *rbacConfig {
		var rules []*Rule
		for _, rule := range mcpauthpolicy.Spec.Rules {
			for _, match := range rule.Matches {
				rbac := &Rule{}

				switch match.Type {
				case v1alpha1.MCPAuthPolicyMatchTypeJWT:
					rbac.Key = match.JWT.Claim
					rbac.Value = match.JWT.Value
				default:
					// TODO: log
					continue
				}

				switch rule.Resource.Kind {
				case "tool":
					rbac.Resource = &Rule_Resource{
						Id:   rule.Resource.Name,
						Type: Rule_Resource_TOOL,
					}
				default:
					// TODO: log
					continue
				}

				rules = append(rules, rbac)
			}
		}

		var gateways []types.NamespacedName
		for _, targetRef := range mcpauthpolicy.Spec.TargetRefs {
			if targetRef.Group != "gateway.networking.k8s.io" && targetRef.Group != "" {
				// TODO: log
				continue
			}
			if targetRef.Kind != "Gateway" && targetRef.Kind != "" {
				// TODO: log
				continue
			}
			gateways = append(gateways, types.NamespacedName{
				Namespace: mcpauthpolicy.Namespace,
				Name:      string(targetRef.Name),
			})
		}

		cfg := &Config{
			Name:      mcpauthpolicy.Name,
			Namespace: mcpauthpolicy.Namespace,
			Rules:     rules,
		}
		cfgName := krt.Named{
			Name:      cfg.GetName(),
			Namespace: cfg.GetNamespace(),
		}.ResourceName()

		return &rbacConfig{
			TargetRefs: gateways,
			Config:     envoyResourceWithCustomName{cfg, cfgName, utils.HashProto(cfg)},
		}
	}, krtopts.ToOptions("mcp-rbac")...)

	mcpGwIndex := krt.NewIndex(mcpRbac, func(rbac rbacConfig) []types.NamespacedName {
		return rbac.TargetRefs
	})

	gateways := krt.NewCollection(s.commonCols.GatewayIndex.Gateways, func(kctx krt.HandlerContext, gw ir.Gateway) *ir.Gateway {
		if gw.Obj.Spec.GatewayClassName != "mcp" {
			return nil
		}
		return &gw

	}, krtopts.ToOptions("mcp-gateways")...)

	////	// translate gateways to xds resources and send it to the mcp relay
	////	gatewaysXds := krt.NewCollection(gateways, func(kctx krt.HandlerContext, gw ir.Gateway) *ir.GatewayIR {
	////		rm := reports.NewReportMap()
	////		r := reports.NewReporter(&rm)
	////		// TODO: don't ignore reports
	////		return s.translator.gwtranslator.Translate(kctx, ctx, &gw, r)
	////	}, krtopts.ToOptions("mcp-gateways")...)

	services := krt.NewManyCollection(s.commonCols.Services, func(kctx krt.HandlerContext, s *corev1.Service) []mcpService {
		var ret []mcpService
		for _, port := range s.Spec.Ports {
			if port.AppProtocol == nil || *port.AppProtocol != "kgateway.dev/mcp" {
				continue
			}
			if s.Spec.ClusterIP == "" && s.Spec.ExternalName == "" {
				continue
			}
			addr := s.Spec.ClusterIP
			if addr == "" {
				addr = s.Spec.ExternalName
			}
			ret = append(ret, mcpService{
				Named: krt.Named{
					Name:      s.Name,
					Namespace: s.Namespace,
				},
				ip:   addr,
				port: int(port.Port),
				path: s.Annotations["kgateway.dev/mcp-path"],
			})
		}
		return ret
	}, krtopts.ToOptions("mcpService")...)

	xdsServices := krt.NewCollection(services, func(kctx krt.HandlerContext, s mcpService) *envoyResourceWithName {
		t := &Target{
			Name: s.ResourceName(),
			Host: s.ip,
			Port: uint32(s.port),
			Path: s.path,
		}
		return &envoyResourceWithName{inner: t, version: utils.HashProto(t)}
	}, krtopts.ToOptions("target-xds")...)

	// translate gateways to xds
	s.xDS = krt.NewCollection(gateways, func(kctx krt.HandlerContext, gw ir.Gateway) *mcpXdsResources {
		gwnn := types.NamespacedName{
			Namespace: gw.Namespace,
			Name:      gw.Name,
		}
		rbacResources := krt.Fetch(kctx, mcpRbac, krt.FilterIndex(mcpGwIndex, gwnn))
		rbacEvnoyResources := make([]envoytypes.Resource, len(rbacResources))
		var rbacVersion uint64
		for i, res := range rbacResources {
			rbacVersion ^= res.Config.version
			rbacEvnoyResources[i] = res.Config
		}

		serviceResources := krt.Fetch(kctx, xdsServices)
		svcEvnoyResources := make([]envoytypes.Resource, len(serviceResources))
		var svcVersion uint64
		for i, res := range serviceResources {
			svcVersion ^= res.version
			svcEvnoyResources[i] = res.inner
		}

		return &mcpXdsResources{
			NamespacedName: types.NamespacedName{
				Namespace: gw.Namespace,
				Name:      gw.Name,
			},
			McpServices: envoycache.NewResources(fmt.Sprintf("%d", svcVersion), svcEvnoyResources),
			McpRbac:     envoycache.NewResources(fmt.Sprintf("%d", rbacVersion), rbacEvnoyResources),
		}
	}, krtopts.ToOptions("mcp-xds")...)
	s.waitForSync = []cache.InformerSynced{
		s.commonCols.HasSynced,
		xdsServices.HasSynced,
		mcpauthpolicies.HasSynced,
		mcpRbac.HasSynced,
		gateways.HasSynced,
		services.HasSynced,
		s.xDS.HasSynced,
	}
	return nil
}

func (s *McpSyncer) Start(ctx context.Context) error {
	logger := contextutils.LoggerFrom(ctx)
	logger.Infof("starting %s Proxy Syncer", s.controllerName)
	// wait for krt collections to sync
	logger.Infof("waiting for cache to sync")
	kube.WaitForCacheSync(
		"kube gw proxy syncer",
		ctx.Done(),
		s.waitForSync...,
	)

	s.xDS.RegisterBatch(func(o []krt.Event[mcpXdsResources], initialSync bool) {
		for _, e := range o {
			if e.Event == controllers.EventDelete {
				// TODO: delete?
				continue
			}
			r := e.Latest()
			snapshot := &mcpSnapshot{
				McpServices: r.McpServices,
				McpRbac:     r.McpRbac,
			}

			s.xdsCache.SetSnapshot(ctx, r.ResourceName(), snapshot)
		}
	}, true)

	return nil
}

type mcpSnapshot struct {
	McpServices envoycache.Resources
	McpRbac     envoycache.Resources
	VersionMap  map[string]map[string]string
}

// GetResources implements cache.ResourceSnapshot.
func (m *mcpSnapshot) GetResources(typeURL string) map[string]envoytypes.Resource {
	resources := m.GetResourcesAndTTL(typeURL)
	if resources == nil {
		return nil
	}

	withoutTTL := make(map[string]envoytypes.Resource, len(resources))

	for k, v := range resources {
		withoutTTL[k] = v.Resource
	}

	return withoutTTL
}

// GetResourcesAndTTL implements cache.ResourceSnapshot.
func (m *mcpSnapshot) GetResourcesAndTTL(typeURL string) map[string]envoytypes.ResourceWithTTL {
	switch typeURL {
	case TargetTypeUrl:
		return m.McpServices.Items
	case TargetConfigUrl:
		return m.McpRbac.Items
	}
	return nil
}

// GetVersion implements cache.ResourceSnapshot.
func (m *mcpSnapshot) GetVersion(typeURL string) string {
	switch typeURL {
	case TargetTypeUrl:
		return m.McpServices.Version
	case TargetConfigUrl:
		return m.McpRbac.Version
	}
	return ""
}

// ConstructVersionMap implements cache.ResourceSnapshot.
func (m *mcpSnapshot) ConstructVersionMap() error {
	if m == nil {
		return fmt.Errorf("missing snapshot")
	}

	resources := map[string]map[string]envoytypes.ResourceWithTTL{
		TargetTypeUrl:   m.McpServices.Items,
		TargetConfigUrl: m.McpRbac.Items,
	}

	// The snapshot resources never change, so no need to ever rebuild.
	if m.VersionMap != nil {
		return nil
	}

	m.VersionMap = make(map[string]map[string]string)
	for typeUrl, items := range resources {
		if _, ok := m.VersionMap[typeUrl]; !ok {
			m.VersionMap[typeUrl] = make(map[string]string, len(items))
		}

		for _, r := range items {
			// Hash our version in here and build the version map.
			marshaledResource, err := envoycache.MarshalResource(r.Resource)
			if err != nil {
				return err
			}
			v := envoycache.HashResource(marshaledResource)
			if v == "" {
				return fmt.Errorf("failed to build resource version: %w", err)
			}

			m.VersionMap[typeUrl][envoycache.GetResourceName(r.Resource)] = v
		}

	}

	return nil
}

// GetVersionMap implements cache.ResourceSnapshot.
func (m *mcpSnapshot) GetVersionMap(typeURL string) map[string]string {
	return m.VersionMap[typeURL]
}

var _ envoycache.ResourceSnapshot = &mcpSnapshot{}

type mcpTranslator struct {
	gwtranslator extensionsplug.KGwTranslator
}

func newTranslator(
	ctx context.Context,
	commonCols *common.CommonCollections,
) *mcpTranslator {

	return &mcpTranslator{
		gwtranslator: gwtranslator.NewTranslator(query.NewData(commonCols)),
	}
}

type report struct {
	// lower case so krt doesn't error in debug handler
	reportMap reports.ReportMap
}

func (r report) ResourceName() string {
	return "report"
}

// do we really need this for a singleton?
func (r report) Equals(in report) bool {
	if !maps.Equal(r.reportMap.Gateways, in.reportMap.Gateways) {
		return false
	}
	if !maps.Equal(r.reportMap.HTTPRoutes, in.reportMap.HTTPRoutes) {
		return false
	}
	if !maps.Equal(r.reportMap.TCPRoutes, in.reportMap.TCPRoutes) {
		return false
	}
	return true
}
