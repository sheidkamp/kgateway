package kgwtest

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/sets"
	"sigs.k8s.io/gateway-api/conformance"
	"sigs.k8s.io/gateway-api/conformance/utils/suite"
	"sigs.k8s.io/gateway-api/pkg/features"
)

// Environment variables consulted by Options when the corresponding field is
// unset. Keeps test binaries flag-free for the common case while still giving
// CI/dev a way to filter.
const (
	EnvRunTest   = "KGWTEST_RUN_TEST"
	EnvSkipTests = "KGWTEST_SKIP_TESTS"
)

// Options configures a new Suite.
type Options struct {
	// GatewayClassName is the kgateway GatewayClass to target. Required.
	GatewayClassName string

	// ManifestFS is the set of filesystems from which Test.Manifests and
	// VersionedSetup manifest paths are resolved.
	ManifestFS []fs.FS

	// SupportedFeatures controls upstream feature-based skipping. If empty,
	// SupportGateway is assumed.
	SupportedFeatures sets.Set[features.FeatureName]

	// VersionedSetup applies one-time suite-scoped fixtures, optionally
	// branching on the detected Gateway API channel/version.
	VersionedSetup VersionedSetup

	// Suite-wide version gating. If unmet, NewSuite returns a Suite that
	// skips every test with a clear reason.
	MinGwApiVersion string
	MaxGwApiVersion string
	RequireChannel  Channel

	// RunTest, if set, filters to a single test by ShortName. Defaults from
	// KGWTEST_RUN_TEST.
	RunTest string

	// SkipTests is a list of ShortNames to skip. Defaults from
	// KGWTEST_SKIP_TESTS (comma-separated).
	SkipTests []string
}

// Suite runs kgwtest.Tests against a shared ConformanceTestSuite.
type Suite struct {
	// Conformance is the underlying gateway-api conformance suite. Exported
	// so test bodies can reach Client, Applier, RoundTripper, TimeoutConfig,
	// etc. — the same handle upstream ConformanceTest.Test receives.
	Conformance *suite.ConformanceTestSuite

	opts       Options
	apiVersion string
	apiChannel Channel
	// suiteSkipReason, if non-empty, makes Run skip every test. Used when
	// suite-wide version bounds aren't satisfied.
	suiteSkipReason string
}

// NewSuite builds a Suite, detects the installed Gateway API version/channel,
// applies the selected VersionedSetup, and returns. If suite-wide version
// bounds are not satisfied, the returned Suite will skip every test in Run
// with a clear reason.
func NewSuite(t *testing.T, opts Options) *Suite {
	t.Helper()
	applyEnvDefaults(&opts)

	confOpts := conformance.DefaultOptions(t)
	confOpts.GatewayClassName = opts.GatewayClassName
	confOpts.ManifestFS = opts.ManifestFS
	if opts.SupportedFeatures != nil && opts.SupportedFeatures.Len() > 0 {
		confOpts.SupportedFeatures = opts.SupportedFeatures
	} else {
		confOpts.SupportedFeatures = sets.New(features.SupportGateway)
	}
	confOpts.SkipTests = append(confOpts.SkipTests, opts.SkipTests...)
	if opts.RunTest != "" {
		confOpts.RunTest = opts.RunTest
	}

	cSuite, err := suite.NewConformanceTestSuite(confOpts)
	require.NoError(t, err, "kgwtest: building conformance suite")

	cSuite.Applier.ManifestFS = cSuite.ManifestFS
	cSuite.Applier.GatewayClass = cSuite.GatewayClassName

	s := &Suite{
		Conformance: cSuite,
		opts:        opts,
	}

	version, channel, err := detectGwApiVersionAndChannel(context.Background(), cSuite.Client)
	require.NoError(t, err, "kgwtest: detecting Gateway API version/channel")
	s.apiVersion = version
	s.apiChannel = channel
	t.Logf("kgwtest: detected Gateway API version=%s channel=%s", version, channel)

	if reason := checkVersionBounds(s.apiVersion, s.apiChannel,
		opts.MinGwApiVersion, opts.MaxGwApiVersion, opts.RequireChannel); reason != "" {
		s.suiteSkipReason = reason
		t.Logf("kgwtest: suite-wide skip: %s", reason)
		return s
	}

	s.applySuiteSetup(t)
	return s
}

// Run executes all tests against this suite. Tests are run as subtests under
// the provided *testing.T, using their ShortName. Parallel tests call
// t.Parallel() internally via Test.Run.
func (s *Suite) Run(t *testing.T, tests []Test) error {
	t.Helper()
	for i := range tests {
		tc := tests[i]
		t.Run(tc.ShortName, func(t *testing.T) {
			if s.suiteSkipReason != "" {
				t.Skipf("kgwtest: %s", s.suiteSkipReason)
			}
			tc.Run(t, s)
		})
	}
	return nil
}

// ApiVersion returns the detected Gateway API bundle version.
func (s *Suite) ApiVersion() string { return s.apiVersion }

// ApiChannel returns the detected Gateway API channel.
func (s *Suite) ApiChannel() Channel { return s.apiChannel }

// shouldSkip evaluates per-test skip conditions: explicit SkipTests list, a
// RunTest filter that names a different test, unmet feature requirements,
// and version/channel bounds.
func (s *Suite) shouldSkip(test *Test) (string, bool) {
	if s.Conformance.SkipTests.Has(test.ShortName) {
		return "test in SkipTests", true
	}
	if s.Conformance.RunTest != "" && s.Conformance.RunTest != test.ShortName {
		return "RunTest filter does not match", true
	}
	for _, f := range test.Features {
		if !s.Conformance.SupportedFeatures.Has(f) {
			return fmt.Sprintf("suite does not support feature %q", f), true
		}
	}
	if reason := checkVersionBounds(s.apiVersion, s.apiChannel,
		test.MinGwApiVersion, test.MaxGwApiVersion, test.RequireChannel); reason != "" {
		return reason, true
	}
	return "", false
}

// applySuiteSetup selects the matching Setup from VersionedSetup and applies
// its manifests once. Cleanup is registered via the conformance Applier, which
// ties it to the parent *testing.T.
func (s *Suite) applySuiteSetup(t *testing.T) {
	t.Helper()
	setup := s.opts.VersionedSetup.selectSetup(s.apiVersion, s.apiChannel)
	for _, path := range setup.Manifests {
		s.Conformance.Applier.MustApplyWithCleanup(t, s.Conformance.Client, s.Conformance.TimeoutConfig, path, true)
	}
	if setup.PostApply != nil {
		setup.PostApply(t, s)
	}
}

// applyEnvDefaults fills unset Options fields from environment variables.
func applyEnvDefaults(opts *Options) {
	if opts.RunTest == "" {
		opts.RunTest = os.Getenv(EnvRunTest)
	}
	if len(opts.SkipTests) == 0 {
		if v := os.Getenv(EnvSkipTests); v != "" {
			for _, s := range strings.Split(v, ",") {
				s = strings.TrimSpace(s)
				if s != "" {
					opts.SkipTests = append(opts.SkipTests, s)
				}
			}
		}
	}
}
