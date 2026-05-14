//go:build e2e

package header_modifiers

import (
	"fmt"
	"path/filepath"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kgateway-dev/kgateway/v2/pkg/utils/fsutils"
	"github.com/kgateway-dev/kgateway/v2/pkg/utils/kubeutils/kubectl"
	testdefaults "github.com/kgateway-dev/kgateway/v2/test/e2e/defaults"
	"github.com/kgateway-dev/kgateway/v2/test/e2e/tests/base"
)

const (
	// SuitePrefix names every header_modifiers-owned resource. The suite
	// shares kgateway-base with other suites so prefixed names prevent
	// collisions under SuiteParallel.
	SuitePrefix = "headermods"

	// GatewayName/GatewayNamespace identify the suite-owned Gateway. The
	// suite cannot share common.BaseGateway because TestGatewayLevelHeaderModifiers
	// attaches a TrafficPolicy at the Gateway scope.
	GatewayName      = "headermods-gateway"
	GatewayNamespace = "kgateway-base"

	httpbinName = "headermods-httpbin"
	curlPodName = "headermods-curl"
)

var (
	// Manifests.
	commonManifest                                       = filepath.Join(fsutils.MustGetThisDir(), "testdata", "common.yaml")
	setupWithListenerSetsManifest                        = filepath.Join(fsutils.MustGetThisDir(), "testdata", "setup-with-listenersets.yaml")
	setupManifest                                        = filepath.Join(fsutils.MustGetThisDir(), "testdata", "setup.yaml")
	httpbinManifest                                      = filepath.Join(fsutils.MustGetThisDir(), "testdata", "httpbin.yaml")
	curlManifest                                         = filepath.Join(fsutils.MustGetThisDir(), "testdata", "curl.yaml")
	headerModifiersRouteTrafficPolicyManifest            = filepath.Join(fsutils.MustGetThisDir(), "testdata", "header-modifiers-route.yaml")
	headerModifiersRouteListenerSetTrafficPolicyManifest = filepath.Join(fsutils.MustGetThisDir(), "testdata", "header-modifiers-route-ls.yaml")
	headerModifiersGwTrafficPolicyManifest               = filepath.Join(fsutils.MustGetThisDir(), "testdata", "header-modifiers-gw.yaml")
	headerModifiersLsTrafficPolicyManifest               = filepath.Join(fsutils.MustGetThisDir(), "testdata", "header-modifiers-ls.yaml")
	headerModifiersFromSecretManifest                    = filepath.Join(fsutils.MustGetThisDir(), "testdata", "header-modifiers-from-secret.yaml")

	// proxyObjectMeta locates the Deployment/Service that the kgateway deployer
	// provisions for the suite-owned Gateway.
	proxyObjectMeta = metav1.ObjectMeta{
		Name:      GatewayName,
		Namespace: GatewayNamespace,
	}

	// curlPodExecOpt selects the suite-owned curl pod for kubectl-exec curls.
	curlPodExecOpt = kubectl.PodExecOptions{
		Name:      curlPodName,
		Namespace: GatewayNamespace,
		Container: "curl",
	}

	httpbinLabelSelector = fmt.Sprintf("%s=%s", testdefaults.WellKnownAppLabel, httpbinName)
	curlPodLabelSelector = fmt.Sprintf("%s=%s", testdefaults.WellKnownAppLabel, curlPodName)

	commonSetupManifests = []string{commonManifest, httpbinManifest, curlManifest}

	setup = base.TestCase{
		Manifests: append([]string{setupManifest}, commonSetupManifests...),
	}

	setupWithListenerSets = base.TestCase{
		Manifests: append([]string{}, commonSetupManifests...),
		ManifestsWithTransform: map[string]func(string) string{
			setupWithListenerSetsManifest: base.TransformListenerSetManifest,
		},
	}

	testCases = map[string]*base.TestCase{
		"TestRouteLevelHeaderModifiers": {
			Manifests: []string{headerModifiersRouteTrafficPolicyManifest},
		},
		"TestGatewayLevelHeaderModifiers": {
			Manifests: []string{headerModifiersGwTrafficPolicyManifest},
		},
		"TestMultiLevelHeaderModifiers": {
			Manifests: []string{
				headerModifiersGwTrafficPolicyManifest,
				headerModifiersRouteTrafficPolicyManifest,
			},
			ManifestsWithTransform: map[string]func(string) string{
				headerModifiersLsTrafficPolicyManifest: base.TransformListenerSetManifest,
			},
		},
		"TestMultiLevelHeaderModifiersWithListenerSet": {
			Manifests: []string{
				headerModifiersGwTrafficPolicyManifest,
				headerModifiersRouteTrafficPolicyManifest,
				headerModifiersRouteListenerSetTrafficPolicyManifest,
			},
			ManifestsWithTransform: map[string]func(string) string{
				headerModifiersLsTrafficPolicyManifest: base.TransformListenerSetManifest,
			},
			MinGwApiVersion: base.GwApiRequireListenerSets,
		},
		"TestListenerSetLevelHeaderModifiers": {
			ManifestsWithTransform: map[string]func(string) string{
				headerModifiersLsTrafficPolicyManifest: base.TransformListenerSetManifest,
			},
			MinGwApiVersion: base.GwApiRequireListenerSets,
		},
		"TestHeaderModifiersFromSecret": {
			Manifests: []string{headerModifiersFromSecretManifest},
		},
	}
)
