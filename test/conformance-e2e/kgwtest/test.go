// Package kgwtest provides a kgateway-specific wrapper around the gateway-api
// conformance suite. See test/conformance-e2e/README for usage.
package kgwtest

import (
	"testing"

	"sigs.k8s.io/gateway-api/conformance/utils/suite"
	"sigs.k8s.io/gateway-api/pkg/features"
)

// Channel represents a Gateway API CRD channel.
type Channel string

const (
	ChannelStandard     Channel = "standard"
	ChannelExperimental Channel = "experimental"
)

// Test describes a single kgateway test scenario.
type Test struct {
	// ShortName uniquely identifies the test and drives filtering and the
	// default per-test namespace name.
	ShortName string

	// Description is a human-readable summary shown in logs.
	Description string

	// Manifests is a list of paths (resolved against Suite.ManifestFS) that
	// are rendered through text/template with TestNamespace and
	// GatewayClassName and then applied before Test runs. Resources are
	// cleaned up automatically via t.Cleanup.
	Manifests []string

	// Parallel, when true, causes the subtest to call t.Parallel().
	Parallel bool

	// Slow flags the test as slow for reporting purposes.
	Slow bool

	// Features lists upstream Gateway API features this test exercises.
	// Skipped if the suite has not opted into the listed features.
	Features []features.FeatureName

	// Provisional marks tests that are still being stabilized.
	Provisional bool

	// MinGwApiVersion is the minimum Gateway API semver required to run this
	// test. Empty means no lower bound. Example: "1.3.0".
	MinGwApiVersion string

	// MaxGwApiVersion is the maximum Gateway API semver for this test.
	// Empty means no upper bound.
	MaxGwApiVersion string

	// RequireChannel, if set, skips the test when the detected channel does
	// not match.
	RequireChannel Channel

	// Namespace overrides the default per-test namespace name. Default is
	// "kgw-e2e-<slug(ShortName)>".
	Namespace string

	// Test is the test body. It receives a TestContext carrying the
	// per-test namespace and the underlying ConformanceTestSuite.
	Test func(t *testing.T, ctx TestContext)
}

// TestContext is the handle passed to each Test body. It exposes the per-test
// namespace along with the underlying ConformanceTestSuite so assertions have
// access to Client, RoundTripper, TimeoutConfig, etc.
type TestContext struct {
	// Suite is the gateway-api conformance suite handle.
	Suite *suite.ConformanceTestSuite

	// Namespace is the per-test namespace created by the framework.
	// All templated manifests are applied into this namespace.
	Namespace string
}

// Run executes the test against the provided Suite. It handles skip
// filtering, per-test namespace creation, manifest templating, and invokes
// the Test body.
func (test *Test) Run(t *testing.T, s *Suite) {
	t.Helper()

	if test.Parallel {
		t.Parallel()
	}

	if reason, skip := s.shouldSkip(test); skip {
		t.Skipf("kgwtest: skipping %s: %s", test.ShortName, reason)
	}

	ns := s.ensureTestNamespace(t, test)
	s.applyManifestsInNamespace(t, test, ns)
	// Registered after applyManifestsInNamespace so it runs before manifest
	// cleanup (t.Cleanup is LIFO) and can dump live test resources.
	t.Cleanup(func() { failureHook(t, test, ns) })

	if test.Test == nil {
		t.Fatalf("kgwtest: test %q has no Test func", test.ShortName)
	}
	test.Test(t, TestContext{Suite: s.Conformance, Namespace: ns})
}
