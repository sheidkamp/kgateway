//go:build e2e

package basicroutinggatewaywithroute_test

import (
	"embed"
	"io/fs"
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"sigs.k8s.io/gateway-api/conformance"
	httputils "sigs.k8s.io/gateway-api/conformance/utils/http"
	gwconformance "sigs.k8s.io/gateway-api/conformance/utils/kubernetes"
	"sigs.k8s.io/gateway-api/conformance/utils/suite"
	"sigs.k8s.io/gateway-api/pkg/features"
)

const (
	gatewayName      = "gateway"
	gatewayNamespace = "kgateway-standalone-conformance"
	routeName        = "example-route"
	backendName      = "example-svc"
	hostHeader       = "example.com"
)

//go:embed testdata/*.yaml
var manifests embed.FS

func TestGatewayWithRoute(t *testing.T) {
	cSuite := newStandaloneConformanceSuite(t)
	tests := []suite.ConformanceTest{gatewayWithRouteTest}

	applySuiteScopedResources(t, cSuite, []string{
		"testdata/namespace.yaml",
		"testdata/service.yaml",
	})

	err := cSuite.Run(t, tests)
	require.NoError(t, err)
}

func newStandaloneConformanceSuite(t *testing.T) *suite.ConformanceTestSuite {
	t.Helper()

	opts := conformance.DefaultOptions(t)
	opts.GatewayClassName = "kgateway"
	opts.SupportedFeatures = sets.New(features.SupportGateway)
	opts.ManifestFS = []fs.FS{manifests}

	cSuite, err := suite.NewConformanceTestSuite(opts)
	require.NoError(t, err)

	// `ConformanceTest.Run()` applies test manifests through the suite applier.
	cSuite.Applier.ManifestFS = cSuite.ManifestFS
	cSuite.Applier.GatewayClass = cSuite.GatewayClassName

	return cSuite
}

var gatewayWithRouteTest = suite.ConformanceTest{
	ShortName:   "KGatewayStandaloneGatewayWithRoute",
	Description: "Verify a standalone Gateway and HTTPRoute can route traffic on both listeners using the Gateway API conformance harness.",
	Manifests: []string{
		"testdata/gateway-with-route.yaml",
	},
	Test: func(t *testing.T, cSuite *suite.ConformanceTestSuite) {
		controllerName := gwconformance.GWCMustHaveAcceptedConditionTrue(
			t,
			cSuite.Client,
			cSuite.TimeoutConfig,
			cSuite.GatewayClassName,
		)

		gatewayNN := types.NamespacedName{
			Name:      gatewayName,
			Namespace: gatewayNamespace,
		}
		routeNN := types.NamespacedName{
			Name:      routeName,
			Namespace: gatewayNamespace,
		}

		for _, listenerName := range []string{"http", "http-privileged"} {
			t.Run(listenerName, func(t *testing.T) {
				gatewayAddr := gwconformance.GatewayAndHTTPRoutesMustBeAccepted(
					t,
					cSuite.Client,
					cSuite.TimeoutConfig,
					controllerName,
					gwconformance.NewGatewayRef(gatewayNN, listenerName),
					routeNN,
				)

				httputils.MakeRequestAndExpectEventuallyConsistentResponse(
					t,
					cSuite.RoundTripper,
					cSuite.TimeoutConfig,
					gatewayAddr,
					httputils.ExpectedResponse{
						Request: httputils.Request{
							Host: hostHeader,
							Path: "/",
						},
						Response: httputils.Response{
							StatusCodes: []int{200},
						},
						Backend:   backendName,
						Namespace: gatewayNamespace,
					},
				)
			})
		}
	},
}

func applySuiteScopedResources(t *testing.T, cSuite *suite.ConformanceTestSuite, manifests []string) {
	t.Helper()

	for _, manifest := range manifests {
		cSuite.Applier.MustApplyWithCleanup(t, cSuite.Client, cSuite.TimeoutConfig, manifest, true)
	}
}
