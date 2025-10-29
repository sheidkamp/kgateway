//go:build e2e

package agentgateway

import (
	"context"
	"fmt"
	"net/http"

	"github.com/stretchr/testify/suite"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/kgateway-dev/kgateway/v2/api/v1alpha1"
	"github.com/kgateway-dev/kgateway/v2/internal/kgateway/wellknown"
	"github.com/kgateway-dev/kgateway/v2/pkg/utils/kubeutils"
	"github.com/kgateway-dev/kgateway/v2/pkg/utils/requestutils/curl"
	"github.com/kgateway-dev/kgateway/v2/test/e2e"
	"github.com/kgateway-dev/kgateway/v2/test/e2e/defaults"
	"github.com/kgateway-dev/kgateway/v2/test/e2e/tests/base"
	"github.com/kgateway-dev/kgateway/v2/test/gomega/matchers"
	"github.com/kgateway-dev/kgateway/v2/test/testutils"
)

type testingSuite struct {
	*base.BaseTestingSuite
}

func NewTestingSuite(ctx context.Context, testInst *e2e.TestInstallation) suite.TestingSuite {
	baseSuite := base.NewBaseTestingSuite(ctx, testInst, base.TestCase{}, testCases)
	// This suite applies TrafficPolicy to specific named sections of the HTTPRoute, and requires HTTPRoutes.spec.rules[].name to be present in the Gateway API version.
	baseSuite.MinGatewayApiVersion = map[base.GatewayApiChannel]*base.GwApiVersion{
		base.GwApiChannelExperimental: &base.GwApiV1_2_0,
		base.GwApiChannelStandard:     &base.GwApiV1_3_0,
	}
	return &testingSuite{
		BaseTestingSuite: baseSuite,
	}
}

func (s *testingSuite) TestAgentgatewayTCPRoute() {
	s.TestInstallation.Assertions.EventuallyGatewayCondition(
		s.Ctx,
		tcpGatewayObjectMeta.Name,
		tcpGatewayObjectMeta.Namespace,
		gwv1.GatewayConditionProgrammed,
		metav1.ConditionTrue,
	)
	s.TestInstallation.Assertions.EventuallyGatewayCondition(
		s.Ctx,
		tcpGatewayObjectMeta.Name,
		tcpGatewayObjectMeta.Namespace,
		gwv1.GatewayConditionAccepted,
		metav1.ConditionTrue,
	)
	s.TestInstallation.Assertions.EventuallyGatewayListenerAttachedRoutes(
		s.Ctx,
		tcpGatewayObjectMeta.Name,
		tcpGatewayObjectMeta.Namespace,
		"tcp",
		1,
	)

	s.TestInstallation.Assertions.AssertEventualCurlResponse(
		s.Ctx,
		defaults.CurlPodExecOpt,
		[]curl.Option{
			curl.WithHost(kubeutils.ServiceFQDN(tcpGatewayObjectMeta)),
			curl.VerboseOutput(),
			curl.WithPort(8080),
		},
		&matchers.HttpResponse{
			StatusCode: http.StatusOK,
		},
	)
}

func (s *testingSuite) TestAgentgatewayHTTPRoute() {
	// modify the default agentgateway GatewayClass to point to the custom GatewayParameters
	err := s.TestInstallation.Actions.Kubectl().RunCommand(s.Ctx, "patch", "--type", "json",
		"gatewayclass", wellknown.DefaultAgwClassName, "-p",
		fmt.Sprintf(`[{"op": "add", "path": "/spec/parametersRef", "value": {"group":"%s", "kind":"%s", "name":"%s", "namespace":"%s"}}]`,
			v1alpha1.GroupName, wellknown.GatewayParametersGVK.Kind, gatewayParamsObjectMeta.GetName(), gatewayParamsObjectMeta.GetNamespace()))
	s.Require().NoError(err, "patching gatewayclass %s", wellknown.DefaultAgwClassName)

	testutils.Cleanup(s.T(), func() {
		// revert to the original GatewayClass (by removing the parametersRef)
		err := s.TestInstallation.Actions.Kubectl().RunCommand(s.Ctx, "patch", "--type", "json",
			"gatewayclass", wellknown.DefaultAgwClassName, "-p",
			`[{"op": "remove", "path": "/spec/parametersRef"}]`)
		s.Require().NoError(err, "patching gatewayclass %s", wellknown.DefaultAgwClassName)
	})

	s.TestInstallation.Assertions.EventuallyGatewayCondition(
		s.Ctx,
		httpGatewayObjectMeta.Name,
		httpGatewayObjectMeta.Namespace,
		gwv1.GatewayConditionProgrammed,
		metav1.ConditionTrue,
	)
	s.TestInstallation.Assertions.EventuallyGatewayCondition(
		s.Ctx,
		httpGatewayObjectMeta.Name,
		httpGatewayObjectMeta.Namespace,
		gwv1.GatewayConditionAccepted,
		metav1.ConditionTrue,
	)
	s.TestInstallation.Assertions.EventuallyGatewayListenerAttachedRoutes(
		s.Ctx,
		httpGatewayObjectMeta.Name,
		httpGatewayObjectMeta.Namespace,
		"http",
		1,
	)

	s.TestInstallation.Assertions.AssertEventualCurlResponse(
		s.Ctx,
		defaults.CurlPodExecOpt,
		[]curl.Option{
			curl.WithHost(kubeutils.ServiceFQDN(httpGatewayObjectMeta)),
			curl.VerboseOutput(),
			curl.WithHostHeader("www.example.com"),
			curl.WithPath("/status/200"),
			curl.WithPort(8080),
		},
		&matchers.HttpResponse{
			StatusCode: http.StatusOK,
		},
	)
}
