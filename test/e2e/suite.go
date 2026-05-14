//go:build e2e

package e2e

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"
)

// SuiteMode controls whether a suite runs in parallel with its siblings or
// sequentially. The zero value is SuiteSerial so any suite that hasn't been
// explicitly audited and opted in stays on the safe, serial path.
type SuiteMode int

const (
	// SuiteSerial runs the suite sequentially with its siblings. Default.
	SuiteSerial SuiteMode = iota
	// SuiteParallel runs the suite concurrently with its sibling SuiteParallel
	// suites. The author must have confirmed the suite has unique resource
	// names, no cluster-scoped mutation, no kgateway install/uninstall, and
	// no HTTPListenerPolicy attached to a shared Gateway.
	SuiteParallel
)

type (
	NewSuiteFunc func(ctx context.Context, testInstallation *TestInstallation) suite.TestingSuite

	// suiteEntry is the internal representation of a registered suite plus its
	// SuiteEntryOption-driven configuration.
	suiteEntry struct {
		name             string
		newSuite         NewSuiteFunc
		mode             SuiteMode
		gatewayManifests []string
	}

	// SuiteEntryOption configures a registered suite. See WithSuiteMode and
	// WithSuiteGateways.
	SuiteEntryOption func(*suiteEntry)

	orderedSuites struct {
		suites []*suiteEntry
	}

	suites struct {
		suites map[string]*suiteEntry
	}

	// A SuiteRunner is an interface that allows E2E tests to simply Register tests in one location and execute them
	// with Run.
	SuiteRunner interface {
		Run(ctx context.Context, t *testing.T, testInstallation *TestInstallation)
		Register(name string, newSuite NewSuiteFunc)
		RegisterWithOpts(name string, newSuite NewSuiteFunc, opts ...SuiteEntryOption)
	}
)

var (
	_ SuiteRunner = new(orderedSuites)
	_ SuiteRunner = new(suites)
)

// WithSuiteMode sets the suite's execution mode. Default is SuiteSerial.
func WithSuiteMode(mode SuiteMode) SuiteEntryOption {
	return func(e *suiteEntry) {
		e.mode = mode
	}
}

// WithSuiteGateways declares manifest paths for Gateway resources this suite
// depends on. The SuiteRunner collects GatewayManifests across all registered
// suites, de-duplicates by path, and applies each unique path once before any
// suite runs. Suites that share a gateway just reference the same file path;
// suites with a custom gateway list a path in their own testdata/.
func WithSuiteGateways(paths ...string) SuiteEntryOption {
	return func(e *suiteEntry) {
		e.gatewayManifests = append(e.gatewayManifests, paths...)
	}
}

// NewSuiteRunner returns an implementation of TestRunner that will execute tests as specified
// in the ordered parameter.
//
// NOTE: it should be strongly preferred to use unordered tests. Only pass true to this function
// if there is a clear need for the tests to be ordered, and specify in a comment near the call
// to NewSuiteRunner why the tests need to be ordered.
func NewSuiteRunner(ordered bool) SuiteRunner {
	if ordered {
		return new(orderedSuites)
	}

	return new(suites)
}

func newSuiteEntry(name string, newSuite NewSuiteFunc, opts []SuiteEntryOption) *suiteEntry {
	e := &suiteEntry{
		name:     name,
		newSuite: newSuite,
	}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

func (o *orderedSuites) Run(ctx context.Context, t *testing.T, testInstallation *TestInstallation) {
	applyDedupedGatewayManifests(ctx, t, testInstallation, o.suites)
	for _, entry := range o.suites {
		e := entry
		t.Run(e.name, func(t *testing.T) {
			if e.mode == SuiteParallel {
				t.Parallel()
			}
			suite.Run(t, e.newSuite(ctx, testInstallation))
		})
	}
}

func (o *orderedSuites) Register(name string, newSuite NewSuiteFunc) {
	o.RegisterWithOpts(name, newSuite)
}

func (o *orderedSuites) RegisterWithOpts(name string, newSuite NewSuiteFunc, opts ...SuiteEntryOption) {
	if o.suites == nil {
		o.suites = make([]*suiteEntry, 0)
	}
	o.suites = append(o.suites, newSuiteEntry(name, newSuite, opts))
}

func (u *suites) Run(ctx context.Context, t *testing.T, testInstallation *TestInstallation) {
	entries := make([]*suiteEntry, 0, len(u.suites))
	for _, e := range u.suites {
		entries = append(entries, e)
	}
	applyDedupedGatewayManifests(ctx, t, testInstallation, entries)
	// TODO(jbohanon) does some randomness need to be injected here to ensure they aren't run in the same order every time?
	// from https://goplay.tools/snippet/A-qqQCWkFaZ it looks like maps are not stable, but tend toward stability.
	for testName, entry := range u.suites {
		e := entry
		t.Run(testName, func(t *testing.T) {
			if e.mode == SuiteParallel {
				t.Parallel()
			}
			suite.Run(t, e.newSuite(ctx, testInstallation))
		})
	}
}

func (u *suites) Register(name string, newSuite NewSuiteFunc) {
	u.RegisterWithOpts(name, newSuite)
}

func (u *suites) RegisterWithOpts(name string, newSuite NewSuiteFunc, opts ...SuiteEntryOption) {
	if u.suites == nil {
		u.suites = make(map[string]*suiteEntry)
	}
	u.suites[name] = newSuiteEntry(name, newSuite, opts)
}

// applyDedupedGatewayManifests walks the registered suite entries, collects every
// declared GatewayManifest path, de-duplicates by path, and applies each unique
// path once via the TestInstallation's IstioClient. This runs before any suite
// starts so dependent suites can assume their gateways are already present.
func applyDedupedGatewayManifests(_ context.Context, t *testing.T, ti *TestInstallation, entries []*suiteEntry) {
	seen := make(map[string]struct{})
	var paths []string
	for _, e := range entries {
		for _, p := range e.gatewayManifests {
			if _, ok := seen[p]; ok {
				continue
			}
			seen[p] = struct{}{}
			paths = append(paths, p)
		}
	}
	if len(paths) == 0 {
		return
	}
	if err := ti.ClusterContext.IstioClient.ApplyYAMLFiles("", paths...); err != nil {
		t.Fatalf("failed to apply shared gateway manifests: %v", err)
	}
	// TODO: wait for Programmed=True on each Gateway declared in the applied
	// manifests. For now we lean on the existing per-suite "manifests exist"
	// assertions when each suite enters its own SetupSuite.
}
