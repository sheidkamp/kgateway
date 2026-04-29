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

// ManifestTransform mutates raw manifest bytes before they are parsed and
// applied. The Suite is provided so transforms can branch on detected
// Gateway API version/channel or other suite state. A transform that has
// nothing to do should return the input unchanged.
type ManifestTransform func(s *Suite, in []byte) []byte

// Test describes a single kgateway test scenario.
type Test struct {
	// ShortName uniquely identifies the test and drives filtering.
	ShortName string

	// Description is a human-readable summary shown in logs.
	Description string

	// Manifests is a list of paths (resolved against Suite.ManifestFS)
	// applied before Test runs. Resources are cleaned up automatically via
	// t.Cleanup. No templating is performed; manifests must be valid YAML
	// that could be applied directly with kubectl.
	Manifests []string

	// ManifestTransforms are applied in order to the bytes of every
	// Manifests entry before parsing. Use for resource-version compatibility
	// shims (e.g., rewriting a kind name when the API moves between
	// experimental and standard channels). A transform that does not apply
	// to the current suite should return the input unchanged so manifests
	// remain valid for kubectl-based debugging.
	ManifestTransforms []ManifestTransform

	// Labels classifies the test for filtering via Options.RunLabels and
	// Options.SkipLabels. Labels are free-form strings. A test runs if it
	// has at least one label in Options.RunLabels (or RunLabels is empty)
	// and no label in Options.SkipLabels.
	Labels []string

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

	// Namespace, if set, causes the framework to create a namespace with
	// this exact name before the test runs and delete it after. Manifests
	// for this test must hardcode the same namespace name. Two Tests in
	// the same Suite must not share a non-empty Namespace value.
	//
	// Empty (the default) means the test relies on shared namespaces
	// created by VersionedSetup. Authors put resources in those shared
	// namespaces with unique resource names per scenario.
	Namespace string

	// Test is the test body. It receives a TestContext carrying the
	// underlying ConformanceTestSuite and the per-test namespace (if any).
	Test func(t *testing.T, ctx TestContext)
}

// TestContext is the handle passed to each Test body. It exposes the
// underlying ConformanceTestSuite (Client, RoundTripper, TimeoutConfig, etc.)
// and the per-test namespace if Test.Namespace was set.
type TestContext struct {
	// Suite is the gateway-api conformance suite handle.
	Suite *suite.ConformanceTestSuite

	// Namespace is the namespace the framework created for this test, or
	// "" when Test.Namespace was unset (test uses shared namespaces).
	Namespace string
}

// Run executes the test against the provided Suite. It handles skip
// filtering, per-test namespace creation (when Test.Namespace is set),
// manifest application, and invokes the Test body.
func (test *Test) Run(t *testing.T, s *Suite) {
	t.Helper()

	if test.Parallel {
		t.Parallel()
	}

	if reason, skip := s.shouldSkip(test); skip {
		t.Skipf("kgwtest: skipping %s: %s", test.ShortName, reason)
	}

	ns := s.ensureTestNamespace(t, test)
	s.applyManifests(t, test.Manifests, test.ManifestTransforms)
	// Registered after applyManifests so it runs before manifest cleanup
	// (t.Cleanup is LIFO) and can dump live test resources.
	t.Cleanup(func() { failureHook(t, test, ns) })

	if test.Test == nil {
		t.Fatalf("kgwtest: test %q has no Test func", test.ShortName)
	}
	test.Test(t, TestContext{Suite: s.Conformance, Namespace: ns})
}
