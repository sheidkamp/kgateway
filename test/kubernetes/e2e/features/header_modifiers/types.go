//go:build e2e

package header_modifiers

import (
	"path/filepath"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kgateway-dev/kgateway/v2/pkg/utils/fsutils"
	testdefaults "github.com/kgateway-dev/kgateway/v2/test/kubernetes/e2e/defaults"
	"github.com/kgateway-dev/kgateway/v2/test/kubernetes/e2e/tests/base"
)

var (
	// Manifests.
	commonManifest                                       = filepath.Join(fsutils.MustGetThisDir(), "testdata", "common.yaml")
	setupManifest_1_4                                    = filepath.Join(fsutils.MustGetThisDir(), "testdata", "setup-1.4.yaml")
	setupManifest                                        = filepath.Join(fsutils.MustGetThisDir(), "testdata", "setup.yaml")
	headerModifiersRouteTrafficPolicyManifest            = filepath.Join(fsutils.MustGetThisDir(), "testdata", "header-modifiers-route.yaml")
	headerModifiersRouteListenerSetTrafficPolicyManifest = filepath.Join(fsutils.MustGetThisDir(), "testdata", "header-modifiers-route-ls.yaml")
	headerModifiersGwTrafficPolicyManifest               = filepath.Join(fsutils.MustGetThisDir(), "testdata", "header-modifiers-gw.yaml")
	headerModifiersLsTrafficPolicyManifest               = filepath.Join(fsutils.MustGetThisDir(), "testdata", "header-modifiers-ls.yaml")

	proxyObjectMeta = metav1.ObjectMeta{
		Name:      "gw",
		Namespace: "default",
	}

	commonSetupManifests = []string{commonManifest, testdefaults.HttpbinManifest, testdefaults.CurlPodManifest}

	setup = base.TestCase{
		Manifests: append([]string{setupManifest}, commonSetupManifests...),
	}

	setup1_4 = base.TestCase{
		Manifests: append([]string{setupManifest_1_4}, commonSetupManifests...),
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
				headerModifiersLsTrafficPolicyManifest,
				headerModifiersRouteTrafficPolicyManifest,
			},
		},
		"TestMultiLevelHeaderModifiersWithListenerSet": {
			Manifests: []string{
				headerModifiersGwTrafficPolicyManifest,
				headerModifiersLsTrafficPolicyManifest,
				headerModifiersRouteTrafficPolicyManifest,
				headerModifiersRouteListenerSetTrafficPolicyManifest,
			},
			GatewayApiVersion: base.GatewayApiV1_4_0,
		},
		"TestListenerSetLevelHeaderModifiers": {
			Manifests:         []string{headerModifiersLsTrafficPolicyManifest},
			GatewayApiVersion: base.GatewayApiV1_4_0,
		},
	}
)
