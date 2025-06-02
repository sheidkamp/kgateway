package metrics

import (
	"context"
	"net/http"

	"github.com/onsi/gomega"
	"github.com/stretchr/testify/suite"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kgateway-dev/kgateway/v2/pkg/utils/kubeutils"
	"github.com/kgateway-dev/kgateway/v2/pkg/utils/requestutils/curl"
	testmatchers "github.com/kgateway-dev/kgateway/v2/test/gomega/matchers"
	"github.com/kgateway-dev/kgateway/v2/test/kubernetes/e2e"
	testdefaults "github.com/kgateway-dev/kgateway/v2/test/kubernetes/e2e/defaults"
)

var _ e2e.NewSuiteFunc = NewTestingSuite

// testingSuite is a suite of basic control plane metrics.
type testingSuite struct {
	suite.Suite
	ctx              context.Context
	testInstallation *e2e.TestInstallation
}

// NewTestingSuite creates a new testing suite for control plane metrics.
func NewTestingSuite(ctx context.Context, testInst *e2e.TestInstallation) suite.TestingSuite {
	return &testingSuite{
		ctx:              ctx,
		testInstallation: testInst,
	}
}

func (s *testingSuite) TestMetrics() {
	manifests := []string{
		testdefaults.CurlPodManifest,
		exampleServiceManifest,
		gatewayWithRouteManifest,
	}
	manifestObjects := []client.Object{
		testdefaults.CurlPod, // curl
		nginxPod, exampleSvc, // nginx
		proxyService, proxyServiceAccount, proxyDeployment, // proxy
		kgatewayService, // kgateway
	}

	s.T().Cleanup(func() {
		for _, manifest := range manifests {
			err := s.testInstallation.Actions.Kubectl().DeleteFileSafe(s.ctx, manifest)
			s.Require().NoError(err)
		}
		s.testInstallation.Assertions.EventuallyObjectsNotExist(s.ctx, manifestObjects...)
	})

	for _, manifest := range manifests {
		err := s.testInstallation.Actions.Kubectl().ApplyFile(s.ctx, manifest)
		s.Require().NoError(err)
	}
	s.testInstallation.Assertions.EventuallyObjectsExist(s.ctx, manifestObjects...)

	// make sure pods are running
	s.testInstallation.Assertions.EventuallyPodsRunning(s.ctx, testdefaults.CurlPod.GetNamespace(), metav1.ListOptions{
		LabelSelector: testdefaults.CurlPodLabelSelector,
	})
	s.testInstallation.Assertions.EventuallyPodsRunning(s.ctx, nginxPod.ObjectMeta.GetNamespace(), metav1.ListOptions{
		LabelSelector: "app.kubernetes.io/name=nginx",
	})
	s.testInstallation.Assertions.EventuallyPodsRunning(s.ctx, proxyObjectMeta.GetNamespace(), metav1.ListOptions{
		LabelSelector: "app.kubernetes.io/name=gw",
	})
	s.testInstallation.Assertions.EventuallyPodsRunning(s.ctx, kgatewayObjectMeta.GetNamespace(), metav1.ListOptions{
		LabelSelector: "app.kubernetes.io/name=kgateway",
	})

	s.testInstallation.Assertions.AssertEventualCurlResponse(
		s.ctx,
		testdefaults.CurlPodExecOpt,
		[]curl.Option{
			curl.WithHost(kubeutils.ServiceFQDN(proxyObjectMeta)),
			curl.WithHostHeader("example.com"),
			curl.WithPort(8080),
		},
		&testmatchers.HttpResponse{
			StatusCode: http.StatusOK,
			Body:       gomega.ContainSubstring(testdefaults.NginxResponse),
		})

	s.testInstallation.Assertions.AssertEventualCurlResponse(
		s.ctx,
		testdefaults.CurlPodExecOpt,
		[]curl.Option{
			curl.WithHost(kubeutils.ServiceFQDN(kgatewayObjectMeta)),
			curl.WithPort(9092),
			curl.WithPath("/metrics"),
		},
		&testmatchers.HttpResponse{
			StatusCode: http.StatusOK,
			Body: gomega.And(
				gomega.MatchRegexp(`kgateway_controller_startups_total\{controller=\"kgateway\.dev\/kgateway\",start_time=\".*\"\} 1`),
				gomega.MatchRegexp(`kgateway_translator_translation_duration_seconds_count\{translator=\"CombinedTranslator\"\} 2`),
				gomega.MatchRegexp(`kgateway_translator_translations_total\{result=\"success\",translator=\"CombinedTranslator\"\} 2`),
				gomega.MatchRegexp(`kgateway_translator_resources\{namespace=\"default\",resource=\"Gateway\",translator=\"GatewayStatusSyncer\"\} 1`),
				gomega.MatchRegexp(`kgateway_translator_resources\{namespace=\"default\",resource=\"HTTPRoute\",translator=\"RouteStatusSyncer\"\} 1`),
				gomega.MatchRegexp(`kgateway_translator_resources\{namespace=\"default\",resource=\"Listener\",translator=\"TranslateProxy\"\} 1`),
				gomega.MatchRegexp(`kgateway_snapshot_syncs_total\{proxy=\"kgateway-.*\",result=\"success\",snapshot=\"ProxyTranslator\"\} 4`),
				gomega.MatchRegexp(`kgateway_snapshot_sync_duration_seconds_count\{proxy=\"kgateway-.*\",snapshot=\"ProxyTranslator\"\} 4`),
			),
		})
}
