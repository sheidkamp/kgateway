package metrics

import (
	"path/filepath"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	apiv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/kgateway-dev/kgateway/v2/pkg/utils/fsutils"
	e2edefaults "github.com/kgateway-dev/kgateway/v2/test/kubernetes/e2e/defaults"
	"github.com/kgateway-dev/kgateway/v2/test/kubernetes/e2e/tests/base"
)

var (
	// manifests
	setupManifest           = filepath.Join(fsutils.MustGetThisDir(), "testdata", "setup.yaml")
	exampleRouteManifest    = filepath.Join(fsutils.MustGetThisDir(), "testdata", "example-route.yaml")
	metricResourcesManifest = filepath.Join(fsutils.MustGetThisDir(), "testdata", "metric-resources.yaml")

	// objects
	proxyObjectMeta = metav1.ObjectMeta{
		Name:      "gw1",
		Namespace: "default",
	}

	proxyDeployment     = &appsv1.Deployment{ObjectMeta: proxyObjectMeta}
	proxyService        = &corev1.Service{ObjectMeta: proxyObjectMeta}
	proxyServiceAccount = &corev1.ServiceAccount{ObjectMeta: proxyObjectMeta}

	kgatewayObjectMeta = metav1.ObjectMeta{
		Name:      "kgateway",
		Namespace: "kgateway-test",
	}

	kgatewayService = &corev1.Service{ObjectMeta: kgatewayObjectMeta}

	exampleSvc = &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "example-svc",
			Namespace: "default",
		},
	}

	exampleRoute = &apiv1.HTTPRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "example-route",
			Namespace: "default",
		},
	}

	nginxPod = &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nginx",
			Namespace: "default",
		},
	}

	setup = base.SimpleTestCase{
		Manifests: []string{setupManifest, e2edefaults.CurlPodManifest},
		Resources: []client.Object{kgatewayService, exampleSvc, proxyDeployment, proxyService, proxyServiceAccount, nginxPod, e2edefaults.CurlPod},
	}

	testCases = map[string]*base.TestCase{
		"TestMetrics": {
			SimpleTestCase: base.SimpleTestCase{
				Manifests: []string{exampleRouteManifest},
				Resources: []client.Object{exampleRoute},
			},
		},
		"TestResourceCountingMetrics": {
			SimpleTestCase: base.SimpleTestCase{
				Manifests: []string{metricResourcesManifest},
				Resources: []client.Object{},
			},
		},
	}
)
