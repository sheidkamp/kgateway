package proxy_syncer

import (
	"context"
	"fmt"

	envoytypes "github.com/envoyproxy/go-control-plane/pkg/cache/types"
	envoycache "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"go.uber.org/atomic"
	"go.uber.org/zap"
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
	"github.com/kgateway-dev/kgateway/v2/internal/kgateway/utils/krtutil"
	"github.com/kgateway-dev/kgateway/v2/internal/kgateway/wellknown"
	"github.com/kgateway-dev/kgateway/v2/internal/kgateway/xds"
)

type McpSyncer struct {
	commonCols     *common.CommonCollections
	translator     *mcpTranslator
	controllerName string

	xDS      krt.Collection[mcpXdsResources]
	xdsCache envoycache.SnapshotCache
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
	return xds.OwnerNamespaceNameID(wellknown.GatewayApiProxyValue, r.Namespace, r.Name)
}

func (r mcpXdsResources) Equals(in mcpXdsResources) bool {
	return r.NamespacedName == in.NamespacedName && report{r.reports}.Equals(report{in.reports}) && r.McpServices.Version == in.McpServices.Version
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
	gateways := krt.NewCollection(s.commonCols.GatewayIndex.Gateways, func(kctx krt.HandlerContext, gw ir.Gateway) *ir.GatewayIR {
		if gw.Obj.Spec.GatewayClassName != "mcp" {
			return nil
		}

		rm := reports.NewReportMap()
		r := reports.NewReporter(&rm)
		// TODO: don't ignore reports
		return s.translator.gwtranslator.Translate(kctx, ctx, &gw, r)
	}, krtopts.ToOptions("mcp-gateways")...)

	services := krt.NewManyCollection(s.commonCols.Services, func(kctx krt.HandlerContext, s *corev1.Service) []mcpService {
		var ret []mcpService
		for _, port := range s.Spec.Ports {
			if corev1.Protocol(*port.AppProtocol) != "kgateway.dev/mcp" {
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
	})
	xds := krt.NewCollection(services, func(kctx krt.HandlerContext, s mcpService) *envoytypes.ResourceWithName {
		panic("TODO")
	})

	version := atomic.NewInt32(0)

	// translate gateways to xds
	s.xDS = krt.NewCollection(gateways, func(kctx krt.HandlerContext, gw ir.GatewayIR) *mcpXdsResources {
		resources := krt.Fetch(kctx, xds)

		r := make([]envoytypes.Resource, len(resources))
		for i, res := range resources {
			r[i] = res
		}

		return &mcpXdsResources{
			NamespacedName: types.NamespacedName{
				Namespace: gw.SourceObject.Namespace,
				Name:      gw.SourceObject.Name,
			},
			McpServices: envoycache.NewResources(fmt.Sprintf("%d", version.Add(1)), r),
		}
	}, krtopts.ToOptions("mcp-xds")...)

	return nil
}

func (s *McpSyncer) Start(ctx context.Context) error {

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

const typeUrl = "type.googleapis.com/ToDo.Type"

type mcpSnapshot struct {
	mcpServices envoycache.Resources

	VersionMap map[string]map[string]string
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

	if _, ok := m.VersionMap[typeUrl]; !ok {
		m.VersionMap[typeUrl] = make(map[string]string, len(m.mcpServices.Items))
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

		m.VersionMap[typeUrl][envoycache.GetResourceName(r.Resource)] = v

	}

	return nil
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
	if typeURL == typeUrl {
		return m.mcpServices.Items
	}
	return nil
}

// GetVersion implements cache.ResourceSnapshot.
func (m *mcpSnapshot) GetVersion(typeURL string) string {
	if typeURL == typeUrl {
		return m.mcpServices.Version
	}
	return ""
}

// GetVersionMap implements cache.ResourceSnapshot.
func (m *mcpSnapshot) GetVersionMap(typeURL string) map[string]string {
	return map[string]string{
		typeUrl: m.mcpServices.Version,
	}
}

var _ envoycache.ResourceSnapshot = &mcpSnapshot{}

type mcpTranslator struct {
	waitForSync []cache.InformerSynced

	gwtranslator extensionsplug.KGwTranslator

	logger *zap.Logger
}

func newTranslator(
	ctx context.Context,
	commonCols *common.CommonCollections,
) *mcpTranslator {

	return &mcpTranslator{
		gwtranslator: gwtranslator.NewTranslator(query.NewData(commonCols)),
	}
}
