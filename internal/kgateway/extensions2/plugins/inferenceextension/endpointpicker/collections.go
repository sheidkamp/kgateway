package endpointpicker

import (
	"context"
	"fmt"

	envoyclusterv3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	skubeclient "istio.io/istio/pkg/config/schema/kubeclient"
	"istio.io/istio/pkg/kube/kclient"
	"istio.io/istio/pkg/kube/krt"
	"istio.io/istio/pkg/kube/kubetypes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	inf "sigs.k8s.io/gateway-api-inference-extension/api/v1"
	"sigs.k8s.io/gateway-api-inference-extension/client-go/clientset/versioned"

	"github.com/kgateway-dev/kgateway/v2/internal/kgateway/krtcollections"
	"github.com/kgateway-dev/kgateway/v2/internal/kgateway/wellknown"
	"github.com/kgateway-dev/kgateway/v2/pkg/pluginsdk/collections"
	"github.com/kgateway-dev/kgateway/v2/pkg/pluginsdk/ir"
	krtpkg "github.com/kgateway-dev/kgateway/v2/pkg/utils/krtutil"
)

type inferencePoolPlugin struct {
	// Envoy & policies use backendsDP; status uses backendsCtl.
	backendsDP  krt.Collection[ir.BackendObjectIR]
	backendsCtl krt.Collection[ir.BackendObjectIR]
	endpoints   krt.Collection[ir.EndpointsForBackend]
	policies    krt.Collection[ir.PolicyWrapper]
	poolIndex   krt.Index[string, ir.BackendObjectIR]
	podIndex    krt.Index[string, krtcollections.LocalityPod]
}

func registerTypes(cli versioned.Interface) {
	skubeclient.Register[*inf.InferencePool](
		wellknown.InferencePoolGVR,
		wellknown.InferencePoolGVK,
		func(c skubeclient.ClientGetter, namespace string, o metav1.ListOptions) (runtime.Object, error) {
			return cli.InferenceV1().InferencePools(namespace).List(context.Background(), o)
		},
		func(c skubeclient.ClientGetter, namespace string, o metav1.ListOptions) (watch.Interface, error) {
			return cli.InferenceV1().InferencePools(namespace).Watch(context.Background(), o)
		},
		func(c skubeclient.ClientGetter, namespace string) kubetypes.WriteAPI[*inf.InferencePool] {
			return cli.InferenceV1().InferencePools(namespace)
		},
	)
}

func initInferencePoolCollections(
	ctx context.Context,
	commonCol *collections.CommonCollections,
) (*inferencePoolPlugin, kclient.Client[*inf.InferencePool]) {
	// Create the inference extension client
	clientset, err := versioned.NewForConfig(commonCol.Client.RESTConfig())
	if err != nil {
		logger.Error("failed to create inference extension client", "error", err)
		return nil, nil
	}

	// Register the InferencePool type
	registerTypes(clientset)

	// Create an InferencePool krt collection
	cli := kclient.NewFilteredDelayed[*inf.InferencePool](
		commonCol.Client,
		wellknown.InferencePoolGVR,
		kclient.Filter{ObjectFilter: commonCol.Client.ObjectFilter()},
	)
	poolCol := krt.WrapClient(cli, commonCol.KrtOpts.ToOptions("InferencePool")...)

	// Create a krt index of pods whose labels match the InferencePool's selector
	podIdx := krtpkg.UnnamedIndex(
		commonCol.LocalityPods,
		func(p krtcollections.LocalityPod) []string {
			var keys []string
			for _, pool := range poolCol.List() {
				sel := labels.Set(convertSelector(pool.Spec.Selector.MatchLabels))
				if p.Namespace == pool.Namespace &&
					labels.SelectorFromSet(sel).Matches(labels.Set(p.AugmentedLabels)) {
					nn := fmt.Sprintf("%s/%s", pool.Namespace, pool.Name)
					keys = append(keys, nn)
				}
			}
			return keys
		})

	// Controller backends – only the InferencePool drives this collection
	backendsCtl := krt.NewCollection(
		poolCol,
		func(_ krt.HandlerContext, p *inf.InferencePool) *ir.BackendObjectIR {
			irPool := newInferencePool(p)
			if errs := validatePool(p, commonCol.Services); len(errs) > 0 {
				irPool.setErrors(errs)
			}
			return buildBackendObjIrFromPool(irPool)
		},
		commonCol.KrtOpts.ToOptions("InferencePoolBackendsCtl")...,
	)

	// Data‑plane backends – rebuilt on any pod change to update LB endpoints
	backendsDP := krt.NewCollection(
		poolCol,
		func(ctx krt.HandlerContext, ip *inf.InferencePool) *ir.BackendObjectIR {
			irPool := newInferencePool(ip)
			pods := krt.Fetch(ctx, commonCol.LocalityPods, krt.FilterGeneric(func(obj any) bool {
				pod, ok := obj.(krtcollections.LocalityPod)
				if !ok {
					return false
				}
				sel := labels.SelectorFromSet(irPool.podSelector)
				return pod.Namespace == ip.Namespace && sel.Matches(labels.Set(pod.AugmentedLabels))
			}))

			var eps []endpoint

			for _, p := range pods {
				if ip := p.Address(); ip != "" {
					// Note: InferencePool v1 only supports a single port
					eps = append(eps, endpoint{address: ip, port: irPool.targetPorts[0].number})
				}
			}
			if len(eps) == 0 {
				return nil
			}
			irPool.setEndpoints(eps)
			return buildBackendObjIrFromPool(irPool)
		},
		commonCol.KrtOpts.ToOptions("InferencePoolBackendsDP")...,
	)

	// Build a static + subset LB cluster per InferencePool
	endpoints := krt.NewCollection(
		backendsDP,
		func(_ krt.HandlerContext, be ir.BackendObjectIR) *ir.EndpointsForBackend {
			stub := &envoyclusterv3.Cluster{Name: be.ClusterName()}
			return processPoolBackendObjIR(ctx, be, stub, podIdx)
		},
	)

	// Index pools by NamespacedName for status management & policy wiring
	poolIdx := krtpkg.UnnamedIndex(backendsCtl, func(be ir.BackendObjectIR) []string {
		return []string{be.ResourceName()}
	})

	// Build a PolicyWrapper collection for the per-route metadata filter
	// and ext-proc overrides.
	policies := buildPolicyWrapperCollection(commonCol, backendsDP)

	return &inferencePoolPlugin{
		backendsDP:  backendsDP,
		backendsCtl: backendsCtl,
		endpoints:   endpoints,
		policies:    policies,
		poolIndex:   poolIdx,
		podIndex:    podIdx,
	}, cli
}
