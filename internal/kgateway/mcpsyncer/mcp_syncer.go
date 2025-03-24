package mcpsyncer

//go:generate go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28
//go:generate protoc --proto_path=. --go_out=. --go_opt=paths=source_relative target.proto

import (
	"context"
	"fmt"
	"maps"

	envoytypes "github.com/envoyproxy/go-control-plane/pkg/cache/types"
	envoycache "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/solo-io/go-utils/contextutils"
	"istio.io/istio/pkg/kube"
	"istio.io/istio/pkg/kube/controllers"
	"istio.io/istio/pkg/kube/krt"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/kgateway-dev/kgateway/v2/internal/kgateway/extensions2/common"
	extensionsplug "github.com/kgateway-dev/kgateway/v2/internal/kgateway/extensions2/plugin"
	"github.com/kgateway-dev/kgateway/v2/internal/kgateway/ir"
	"github.com/kgateway-dev/kgateway/v2/internal/kgateway/query"
	"github.com/kgateway-dev/kgateway/v2/internal/kgateway/reports"
	gwtranslator "github.com/kgateway-dev/kgateway/v2/internal/kgateway/translator/gateway"
	"github.com/kgateway-dev/kgateway/v2/internal/kgateway/utils"
	"github.com/kgateway-dev/kgateway/v2/internal/kgateway/utils/krtutil"
	"github.com/kgateway-dev/kgateway/v2/internal/kgateway/xds"
)

const (
	TargetTypeUrl = "type.googleapis.com/mcp.kgateway.dev.target.v1alpha1.Target"
)

type McpSyncer struct {
	commonCols     *common.CommonCollections
	translator     *mcpTranslator
	controllerName string

	xDS      krt.Collection[mcpXdsResources]
	xdsCache envoycache.SnapshotCache

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
	return &McpSyncer{
		commonCols:     commonCols,
		translator:     newTranslator(ctx, commonCols),
		controllerName: controllerName,
		xdsCache:       xdsCache,
		// mgr:            mgr,
		// istioClient:    client,
	}
}

type mcpXdsResources struct {
	types.NamespacedName

	reports     reports.ReportMap
	McpServices envoycache.Resources
}

func (r mcpXdsResources) ResourceName() string {
	return xds.OwnerNamespaceNameID("mcp-kgateway-kube-gateway-api", r.Namespace, r.Name)
}

func (r mcpXdsResources) Equals(in mcpXdsResources) bool {
	return r.NamespacedName == in.NamespacedName && report{r.reports}.Equals(report{in.reports}) && r.McpServices.Version == in.McpServices.Version
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

type mcpService struct {
	krt.Named
	ip   string
	port int
}

func (r mcpService) Equals(in mcpService) bool {
	return r.ip == in.ip && r.port == in.port
}

func (s *McpSyncer) Init(ctx context.Context, krtopts krtutil.KrtOptions) error {
	// find mcp gateways
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
			if s.Spec.ClusterIP == "" {
				continue
			}
			ret = append(ret, mcpService{
				Named: krt.Named{
					Name:      s.Name,
					Namespace: s.Namespace,
				},
				ip:   s.Spec.ClusterIP,
				port: int(port.Port),
			})
		}
		return ret
	}, krtopts.ToOptions("mcpService")...)

	xds := krt.NewCollection(services, func(kctx krt.HandlerContext, s mcpService) *envoyResourceWithName {
		t := &Target{
			Name: s.ResourceName(),
			Host: s.ip,
			Port: uint32(s.port),
		}
		return &envoyResourceWithName{inner: t, version: utils.HashProto(t)}
	}, krtopts.ToOptions("target-xds")...)

	// translate gateways to xds
	s.xDS = krt.NewCollection(gateways, func(kctx krt.HandlerContext, gw ir.Gateway) *mcpXdsResources {
		resources := krt.Fetch(kctx, xds)

		r := make([]envoytypes.Resource, len(resources))
		var version uint64
		for i, res := range resources {
			version ^= res.version
			r[i] = res.inner
		}

		return &mcpXdsResources{
			NamespacedName: types.NamespacedName{
				Namespace: gw.Namespace,
				Name:      gw.Name,
			},
			McpServices: envoycache.NewResources(fmt.Sprintf("%d", version), r),
		}
	}, krtopts.ToOptions("mcp-xds")...)
	s.waitForSync = []cache.InformerSynced{
		s.commonCols.HasSynced,
		xds.HasSynced,
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
				mcpServices: r.McpServices,
			}

			s.xdsCache.SetSnapshot(ctx, r.ResourceName(), snapshot)
		}
	}, true)

	return nil
}

type mcpSnapshot struct {
	mcpServices envoycache.Resources

	VersionMap map[string]map[string]string
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
	if typeURL == TargetTypeUrl {
		return m.mcpServices.Items
	}
	return nil
}

// GetVersion implements cache.ResourceSnapshot.
func (m *mcpSnapshot) GetVersion(typeURL string) string {
	if typeURL == TargetTypeUrl {
		return m.mcpServices.Version
	}
	return ""
}

// ConstructVersionMap implements cache.ResourceSnapshot.
func (m *mcpSnapshot) ConstructVersionMap() error {
	if m == nil {
		return fmt.Errorf("missing snapshot")
	}

	// The snapshot resources never change, so no need to ever rebuild.
	if m.VersionMap != nil {
		return nil
	}

	m.VersionMap = make(map[string]map[string]string)

	if _, ok := m.VersionMap[TargetTypeUrl]; !ok {
		m.VersionMap[TargetTypeUrl] = make(map[string]string, len(m.mcpServices.Items))
	}

	for _, r := range m.mcpServices.Items {
		// Hash our version in here and build the version map.
		marshaledResource, err := envoycache.MarshalResource(r.Resource)
		if err != nil {
			return err
		}
		v := envoycache.HashResource(marshaledResource)
		if v == "" {
			return fmt.Errorf("failed to build resource version: %w", err)
		}

		m.VersionMap[TargetTypeUrl][envoycache.GetResourceName(r.Resource)] = v

	}

	return nil
}

// GetVersionMap implements cache.ResourceSnapshot.
func (m *mcpSnapshot) GetVersionMap(typeURL string) map[string]string {
	return m.VersionMap[TargetTypeUrl]
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
