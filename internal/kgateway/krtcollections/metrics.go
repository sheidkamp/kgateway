package krtcollections

import (
	"istio.io/istio/pkg/kube/controllers"
	"istio.io/istio/pkg/kube/krt"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"
	gwv1a2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
	gwxv1a1 "sigs.k8s.io/gateway-api/apisx/v1alpha1"

	tmetrics "github.com/kgateway-dev/kgateway/v2/internal/kgateway/translator/metrics"
	"github.com/kgateway-dev/kgateway/v2/pkg/metrics"
)

const (
	resourcesSubsystem = "resources"
)

var (
	resourcesManaged = metrics.NewGauge(
		metrics.GaugeOpts{
			Subsystem: resourcesSubsystem,
			Name:      "managed",
			Help:      "Current number of gateway resources managed",
		},
		[]string{"namespace", "parent", "resource"},
	)
)

type resourceMetricLabels struct {
	Namespace string
	Parent    string
	Resource  string
}

func (r resourceMetricLabels) toMetricsLabels() []metrics.Label {
	return []metrics.Label{
		{Name: "namespace", Value: r.Namespace},
		{Name: "parent", Value: r.Parent},
		{Name: "resource", Value: r.Resource},
	}
}

// GetResourceMetricEventHandler returns a function that handles krt events for various Gateway API resources.
func GetResourceMetricEventHandler[T any]() func(krt.Event[T]) {
	var (
		gatewayNames []string
		eventType    controllers.EventType
		namespace    string
		resourceType string
		clientObject any
	)

	return func(o krt.Event[T]) {
		clientObject = o.Latest()
		switch obj := clientObject.(type) {
		case *gwv1.HTTPRoute:
			eventType = o.Event
			resourceType = "HTTPRoute"
			namespace = clientObject.(client.Object).GetNamespace()
			gatewayNames = make([]string, 0, len(obj.Spec.ParentRefs))
			for _, pr := range obj.Spec.ParentRefs {
				gatewayNames = append(gatewayNames, string(pr.Name))
			}
		case *gwv1a2.TCPRoute:
			eventType = o.Event
			resourceType = "TCPRoute"
			namespace = clientObject.(client.Object).GetNamespace()
			gatewayNames = make([]string, 0, len(obj.Spec.ParentRefs))
			for _, pr := range obj.Spec.ParentRefs {
				gatewayNames = append(gatewayNames, string(pr.Name))
			}
		case *gwv1a2.TLSRoute:
			eventType = o.Event
			resourceType = "TLSRoute"
			namespace = clientObject.(client.Object).GetNamespace()
			gatewayNames = make([]string, 0, len(obj.Spec.ParentRefs))
			for _, pr := range obj.Spec.ParentRefs {
				gatewayNames = append(gatewayNames, string(pr.Name))
			}
		case *gwv1.GRPCRoute:
			eventType = o.Event
			resourceType = "GRPCRoute"
			namespace = clientObject.(client.Object).GetNamespace()
			gatewayNames = make([]string, 0, len(obj.Spec.ParentRefs))
			for _, pr := range obj.Spec.ParentRefs {
				gatewayNames = append(gatewayNames, string(pr.Name))
			}
		case *gwv1.Gateway:
			eventType = o.Event
			resourceType = "Gateway"
			namespace = clientObject.(client.Object).GetNamespace()
			gatewayNames = []string{clientObject.(client.Object).GetName()}
		case *gwxv1a1.XListenerSet:
			eventType = o.Event
			resourceType = "XListenerSet"
			namespace = clientObject.(client.Object).GetNamespace()
			gatewayNames = []string{string(obj.Spec.ParentRef.Name)}

			if eventType != controllers.EventDelete {
				tmetrics.StartResourceSync(clientObject.(client.Object).GetName(),
					tmetrics.ResourceMetricLabels{
						Namespace: namespace,
						Gateway:   gatewayNames[0],
						Resource:  resourceType,
					})
			}
		default:
			return
		}

		switch eventType {
		case controllers.EventAdd:
			for _, gatewayName := range gatewayNames {
				resourcesManaged.Add(1, resourceMetricLabels{
					Parent:    gatewayName,
					Namespace: namespace,
					Resource:  resourceType,
				}.toMetricsLabels()...)
			}
		case controllers.EventDelete:
			for _, gatewayName := range gatewayNames {
				resourcesManaged.Sub(1, resourceMetricLabels{
					Parent:    gatewayName,
					Namespace: namespace,
					Resource:  resourceType,
				}.toMetricsLabels()...)
			}
		}
	}
}

// ResetMetrics resets the metrics from this package.
// This is provided for testing purposes only.
func ResetMetrics() {
	resourcesManaged.Reset()
}
