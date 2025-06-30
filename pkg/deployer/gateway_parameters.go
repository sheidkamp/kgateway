package deployer

import (
	"istio.io/api/annotation"
	"istio.io/api/label"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	api "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/kgateway-dev/kgateway/v2/api/v1alpha1"
	common "github.com/kgateway-dev/kgateway/v2/pkg/pluginsdk/collections"
)

// Inputs is the set of options used to configure gateway/ineference pool deployment.
type Inputs struct {
	Dev                      bool
	IstioAutoMtlsEnabled     bool
	ControlPlane             ControlPlaneInfo
	ImageInfo                *ImageInfo
	CommonCollections        *common.CommonCollections
	GatewayClassName         string
	WaypointGatewayClassName string
	AgentGatewayClassName    string
}

type ExtraGatewayParameters struct {
	Group     string
	Kind      string
	Object    client.Object
	Generator HelmValuesGenerator
}

// applyFloatingUserId will set the RunAsUser field from all security contexts to null if the floatingUserId field is set
func ApplyFloatingUserId(dstKube *v1alpha1.KubernetesProxyConfig) {
	floatingUserId := dstKube.GetFloatingUserId()
	if floatingUserId == nil || !*floatingUserId {
		return
	}

	// Pod security context
	podSecurityContext := dstKube.GetPodTemplate().GetSecurityContext()
	if podSecurityContext != nil {
		podSecurityContext.RunAsUser = nil
	}

	// Container security contexts
	securityContexts := []*corev1.SecurityContext{
		dstKube.GetEnvoyContainer().GetSecurityContext(),
		dstKube.GetSdsContainer().GetSecurityContext(),
		dstKube.GetIstio().GetIstioProxyContainer().GetSecurityContext(),
		dstKube.GetAiExtension().GetSecurityContext(),
	}

	for _, securityContext := range securityContexts {
		if securityContext != nil {
			securityContext.RunAsUser = nil
		}
	}
}

type GatewayParametersArgs struct {
	gatewayClass *api.GatewayClass
	useLowPorts  bool
	imageInfo    *ImageInfo
}

func NewGatewayParametersArgs() *GatewayParametersArgs {
	return &GatewayParametersArgs{}
}

func (gwpArgs *GatewayParametersArgs) WithUseLowPorts(useLowPorts bool) *GatewayParametersArgs {
	gwpArgs.useLowPorts = useLowPorts
	return gwpArgs
}

func (gwpArgs *GatewayParametersArgs) WithImageInfo(imageInfo *ImageInfo) *GatewayParametersArgs {
	gwpArgs.imageInfo = imageInfo
	return gwpArgs
}

func (gwpArgs *GatewayParametersArgs) WithGatewayClass(gatewayClass *api.GatewayClass) *GatewayParametersArgs {
	gwpArgs.gatewayClass = gatewayClass
	return gwpArgs
}

func (gwpArgs *GatewayParametersArgs) GetGatewayClass() *api.GatewayClass {
	return gwpArgs.gatewayClass
}

// GetInMemoryGatewayParameters returns an in-memory GatewayParameters based on the name of the gateway class.
func GetInMemoryGatewayParameters(gwpArgs *GatewayParametersArgs) *v1alpha1.GatewayParameters {
	switch gwpArgs.gatewayClass.Name {
	case wellknown.WaypointClassName:
		return defaultWaypointGatewayParameters(gwpArgs)
	case wellknown.GatewayClassName:
		return defaultGatewayParameters(gwpArgs)
	case wellknown.AgentGatewayClassName:
		return defaultAgentGatewayParameters(gwpArgs)
// func GetInMemoryGatewayParameters(name string, imageInfo *ImageInfo, gatewayClassName, waypointClassName, agentGatewayClassName string) *v1alpha1.GatewayParameters {
// 	switch name {
// 	case waypointClassName:
// 		return defaultWaypointGatewayParameters(imageInfo)
// 	case gatewayClassName:
// 		return defaultGatewayParameters(imageInfo)
// 	case agentGatewayClassName:
// 		return defaultAgentGatewayParameters(imageInfo)
	default:
		return defaultGatewayParameters(gwpArgs)
	}
}

// defaultAgentGatewayParameters returns an in-memory GatewayParameters with default values
// set for the agentgateway deployment.
func defaultAgentGatewayParameters(gwpArgs *GatewayParametersArgs) *v1alpha1.GatewayParameters {
	gwp := defaultGatewayParameters(gwpArgs)
	gwp.Spec.Kube.AgentGateway.Enabled = ptr.To(true)
	return gwp
}

// defaultWaypointGatewayParameters returns an in-memory GatewayParameters with default values
// set for the waypoint deployment.
func defaultWaypointGatewayParameters(gwpArgs *GatewayParametersArgs) *v1alpha1.GatewayParameters {
	gwp := defaultGatewayParameters(gwpArgs)
	gwp.Spec.Kube.Service.Type = ptr.To(corev1.ServiceTypeClusterIP)

	if gwp.Spec.Kube.PodTemplate == nil {
		gwp.Spec.Kube.PodTemplate = &v1alpha1.Pod{}
	}
	if gwp.Spec.Kube.PodTemplate.ExtraLabels == nil {
		gwp.Spec.Kube.PodTemplate.ExtraLabels = make(map[string]string)
	}
	gwp.Spec.Kube.PodTemplate.ExtraLabels[label.IoIstioDataplaneMode.Name] = "ambient"

	// do not have zTunnel resolve DNS for us - this can cause traffic loops when we're doing
	// outbound based on DNS service entries
	// TODO do we want this on the north-south gateway class as well?
	if gwp.Spec.Kube.PodTemplate.ExtraAnnotations == nil {
		gwp.Spec.Kube.PodTemplate.ExtraAnnotations = make(map[string]string)
	}
	gwp.Spec.Kube.PodTemplate.ExtraAnnotations[annotation.AmbientDnsCapture.Name] = "false"

	return gwp
}

// defaultGatewayParameters returns an in-memory GatewayParameters with the default values
// set for the gateway.
func defaultGatewayParameters(gwpArgs *GatewayParametersArgs) *v1alpha1.GatewayParameters {
	gwp := &v1alpha1.GatewayParameters{
		Spec: v1alpha1.GatewayParametersSpec{
			SelfManaged: nil,
			Kube: &v1alpha1.KubernetesProxyConfig{
				Deployment: &v1alpha1.ProxyDeployment{
					Replicas: ptr.To[uint32](1),
				},
				Service: &v1alpha1.Service{
					Type: (*corev1.ServiceType)(ptr.To(string(corev1.ServiceTypeLoadBalancer))),
				},
				EnvoyContainer: &v1alpha1.EnvoyContainer{
					Bootstrap: &v1alpha1.EnvoyBootstrap{
						LogLevel: ptr.To("info"),
					},
					Image: &v1alpha1.Image{
						Registry:   ptr.To(gwpArgs.imageInfo.Registry),
						Tag:        ptr.To(gwpArgs.imageInfo.Tag),
						Repository: ptr.To(EnvoyWrapperImage),
						PullPolicy: (*corev1.PullPolicy)(ptr.To(gwpArgs.imageInfo.PullPolicy)),
					},
					SecurityContext: &corev1.SecurityContext{
						AllowPrivilegeEscalation: ptr.To(false),
						ReadOnlyRootFilesystem:   ptr.To(true),
						RunAsNonRoot:             ptr.To(true),
						RunAsUser:                ptr.To[int64](10101),
						Capabilities: &corev1.Capabilities{
							Drop: []corev1.Capability{"ALL"},
							Add:  []corev1.Capability{"NET_BIND_SERVICE"},
						},
					},
				},
				Stats: &v1alpha1.StatsConfig{
					Enabled:                 ptr.To(true),
					RoutePrefixRewrite:      ptr.To("/stats/prometheus?usedonly"),
					EnableStatsRoute:        ptr.To(true),
					StatsRoutePrefixRewrite: ptr.To("/stats"),
				},
				SdsContainer: &v1alpha1.SdsContainer{
					Image: &v1alpha1.Image{
						Registry:   ptr.To(gwpArgs.imageInfo.Registry),
						Tag:        ptr.To(gwpArgs.imageInfo.Tag),
						Repository: ptr.To(SdsImage),
						PullPolicy: (*corev1.PullPolicy)(ptr.To(gwpArgs.imageInfo.PullPolicy)),
					},
					Bootstrap: &v1alpha1.SdsBootstrap{
						LogLevel: ptr.To("info"),
					},
				},
				Istio: &v1alpha1.IstioIntegration{
					IstioProxyContainer: &v1alpha1.IstioContainer{
						Image: &v1alpha1.Image{
							Registry:   ptr.To("docker.io/istio"),
							Repository: ptr.To("proxyv2"),
							Tag:        ptr.To("1.22.0"),
							PullPolicy: (*corev1.PullPolicy)(ptr.To(gwpArgs.imageInfo.PullPolicy)),
						},
						LogLevel:              ptr.To("warning"),
						IstioDiscoveryAddress: ptr.To("istiod.istio-system.svc:15012"),
						IstioMetaMeshId:       ptr.To("cluster.local"),
						IstioMetaClusterId:    ptr.To("Kubernetes"),
					},
				},
				AiExtension: &v1alpha1.AiExtension{
					Enabled: ptr.To(false),
					Image: &v1alpha1.Image{
						Repository: ptr.To(KgatewayAIContainerName),
						Registry:   ptr.To(gwpArgs.imageInfo.Registry),
						Tag:        ptr.To(gwpArgs.imageInfo.Tag),
						PullPolicy: (*corev1.PullPolicy)(ptr.To(gwpArgs.imageInfo.PullPolicy)),
					},
				},
				AgentGateway: &v1alpha1.AgentGateway{
					Enabled:  ptr.To(false),
					LogLevel: ptr.To("info"),
					Image: &v1alpha1.Image{
						Registry:   ptr.To(AgentgatewayRegistry),
						Tag:        ptr.To(AgentgatewayDefaultTag),
						Repository: ptr.To(AgentgatewayImage),
						PullPolicy: (*corev1.PullPolicy)(ptr.To(gwpArgs.imageInfo.PullPolicy)),
					},
					SecurityContext: &corev1.SecurityContext{
						AllowPrivilegeEscalation: ptr.To(false),
						ReadOnlyRootFilesystem:   ptr.To(true),
						RunAsNonRoot:             ptr.To(true),
						RunAsUser:                ptr.To[int64](10101),
						Capabilities: &corev1.Capabilities{
							Drop: []corev1.Capability{"ALL"},
							Add:  []corev1.Capability{"NET_BIND_SERVICE"},
						},
					},
				},
			},
		},
	}

	if gwpArgs.useLowPorts {
		gwp.Spec.Kube.PodTemplate = &v1alpha1.Pod{
			SecurityContext: &corev1.PodSecurityContext{
				Sysctls: []corev1.Sysctl{
					{
						Name:  "net.ipv4.ip_unprivileged_port_start",
						Value: "0",
					},
				},
			},
		}
	}
	return gwp
}
