//go:build e2e

package basicrouting_test

import (
	"testing"

	"k8s.io/apimachinery/pkg/types"
	httputils "sigs.k8s.io/gateway-api/conformance/utils/http"
	gwconformance "sigs.k8s.io/gateway-api/conformance/utils/kubernetes"

	"github.com/kgateway-dev/kgateway/v2/test/conformance-e2e/kgwtest"
)

const (
	gatewayName = "gateway"
	routeName   = "example-route"
	backendName = "example-svc"
	hostHeader  = "example.com"
)

func init() {
	tests = append(tests, gatewayWithRouteTest)
}

var gatewayWithRouteTest = kgwtest.Test{
	ShortName:   "GatewayWithRoute",
	Description: "A Gateway with two HTTP listeners and one HTTPRoute routes traffic on both listeners.",
	Manifests:   []string{"testdata/gateway-with-route.yaml"},
	Parallel:    true,
	Test: func(t *testing.T, ctx kgwtest.TestContext) {
		controllerName := gwconformance.GWCMustHaveAcceptedConditionTrue(
			t, ctx.Suite.Client, ctx.Suite.TimeoutConfig, ctx.Suite.GatewayClassName,
		)

		gatewayNN := types.NamespacedName{Name: gatewayName, Namespace: ctx.Namespace}
		routeNN := types.NamespacedName{Name: routeName, Namespace: ctx.Namespace}

		for _, listenerName := range []string{"http", "http-privileged"} {
			t.Run(listenerName, func(t *testing.T) {
				gatewayAddr := gwconformance.GatewayAndHTTPRoutesMustBeAccepted(
					t, ctx.Suite.Client, ctx.Suite.TimeoutConfig, controllerName,
					gwconformance.NewGatewayRef(gatewayNN, listenerName),
					routeNN,
				)

				httputils.MakeRequestAndExpectEventuallyConsistentResponse(
					t, ctx.Suite.RoundTripper, ctx.Suite.TimeoutConfig, gatewayAddr,
					httputils.ExpectedResponse{
						Request: httputils.Request{
							Host: hostHeader,
							Path: "/",
						},
						Response: httputils.Response{
							StatusCodes: []int{200},
						},
						Backend:   backendName,
						Namespace: ctx.Namespace,
					},
				)
			})
		}
	},
}
