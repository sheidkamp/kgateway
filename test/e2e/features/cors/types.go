//go:build e2e

package cors

import (
	"path/filepath"

	"github.com/kgateway-dev/kgateway/v2/pkg/utils/fsutils"
	"github.com/kgateway-dev/kgateway/v2/test/e2e/tests/base"
)

const (
	// SuitePrefix is the prefix all CORS suite-owned resources share. The
	// CORS suite lives alongside other suites in the shared kgateway-base
	// namespace, so every resource it creates must be uniquely named.
	SuitePrefix = "cors"

	// GatewayName is the suite-owned Gateway. CORS needs a private gateway
	// because TestTrafficPolicyCorsAtGatewayLevel/TestTrafficPolicyRouteCorsOverrideGwCors
	// attach a TrafficPolicy at the Gateway scope, which would conflict with
	// any other suite sharing the same Gateway.
	GatewayName      = "cors-gateway"
	GatewayNamespace = "kgateway-base"
)

var (
	// GatewayManifest declares the suite-owned Gateway. It is surfaced to the
	// run-start gateway loader via WithSuiteGateways so it is applied (and
	// Programmed) before any suite runs.
	GatewayManifest = filepath.Join(fsutils.MustGetThisDir(), "testdata", "gateway.yaml")

	// manifests
	simpleServiceManifest  = filepath.Join(fsutils.MustGetThisDir(), "testdata", "service.yaml")
	httpRoutesManifest     = filepath.Join(fsutils.MustGetThisDir(), "testdata", "httproutes.yaml")
	corsHttpRoutesManifest = filepath.Join(fsutils.MustGetThisDir(), "testdata", "httproutes-cors.yaml")

	// traffic policies with cors configuration
	gwCorsTrafficPolicyManifest    = filepath.Join(fsutils.MustGetThisDir(), "testdata", "tp-gw-cors.yaml")
	routeCorsTrafficPolicyManifest = filepath.Join(fsutils.MustGetThisDir(), "testdata", "tp-route-cors.yaml")

	setup = base.TestCase{
		Manifests: []string{
			simpleServiceManifest,
		},
	}

	testCases = map[string]*base.TestCase{
		"TestTrafficPolicyCorsForRoute": {
			Manifests: []string{httpRoutesManifest, routeCorsTrafficPolicyManifest},
		},
		"TestTrafficPolicyCorsAtGatewayLevel": {
			Manifests: []string{httpRoutesManifest, gwCorsTrafficPolicyManifest},
		},
		"TestTrafficPolicyRouteCorsOverrideGwCors": {
			Manifests: []string{httpRoutesManifest, gwCorsTrafficPolicyManifest, routeCorsTrafficPolicyManifest},
		},
		"TestHttpRouteCorsInRouteRules": {
			Manifests: []string{corsHttpRoutesManifest},
		},
		"TestHttpRouteAndTrafficPolicyCors": {
			Manifests:       []string{corsHttpRoutesManifest, gwCorsTrafficPolicyManifest},
			MinGwApiVersion: base.GwApiRequireCorsFilters,
		},
	}
)
