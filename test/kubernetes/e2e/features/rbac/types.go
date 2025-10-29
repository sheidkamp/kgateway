//go:build e2e

package rbac

import (
	"net/http"
	"path/filepath"

	"github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kgateway-dev/kgateway/v2/pkg/utils/fsutils"
	"github.com/kgateway-dev/kgateway/v2/test/gomega/matchers"
	"github.com/kgateway-dev/kgateway/v2/test/kubernetes/e2e/defaults"
	"github.com/kgateway-dev/kgateway/v2/test/kubernetes/e2e/tests/base"
)

var (
	// manifests
	setupManifest            = filepath.Join(fsutils.MustGetThisDir(), "testdata", "setup.yaml")
	rbacManifest             = filepath.Join(fsutils.MustGetThisDir(), "testdata", "cel-rbac.yaml")
	rbacManifestWithSections = filepath.Join(fsutils.MustGetThisDir(), "testdata", "cel-rbac-section.yaml")
	// Core infrastructure objects that we need to track
	gatewayObjectMeta = metav1.ObjectMeta{
		Name:      "gw",
		Namespace: "default",
	}
	gatewayService = &corev1.Service{ObjectMeta: gatewayObjectMeta}

	expectStatus200Success = &matchers.HttpResponse{
		StatusCode: http.StatusOK,
		Body:       nil,
	}
	expectRBACDenied = &matchers.HttpResponse{
		StatusCode: http.StatusForbidden,
		Body:       gomega.ContainSubstring("RBAC: access denied"),
	}

	commonSetupManifests = []string{defaults.HttpbinManifest, defaults.CurlPodManifest}
	// Base test setup - common infrastructure for all tests
	setup = base.TestCase{
		Manifests: append([]string{setupManifest}, commonSetupManifests...),
	}

	// Individual test cases - test-specific manifests and resources
	testCases = map[string]*base.TestCase{
		"TestRBACHeaderAuthorizationWithRouteLevelRBAC": {
			Manifests: []string{rbacManifestWithSections},
			MinGatewayApiVersion: map[base.GatewayApiChannel]*base.GwApiVersion{
				base.GwApiChannelExperimental: &base.GwApiV1_2_0, // HTTPRoutes.spec.rules[].name was added in 1.2 experimental
				base.GwApiChannelStandard:     &base.GwApiV1_3_0, // HTTPRoutes.spec.rules[].name was added to standard in 1.3
			},
		},
		"TestRBACHeaderAuthorization": {
			Manifests: []string{rbacManifest},
		},
	}
)
