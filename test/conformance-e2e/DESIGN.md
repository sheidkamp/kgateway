# kgateway e2e: base suite on top of gateway-api conformance

## Context

The existing `test/e2e` framework is built around a `TestInstallation` god object ([test/e2e/test.go:98-128](test/e2e/test.go#L98-L128)) that fuses cluster metadata, helm install state, kubectl/istio action providers, assertions, and generated-files management into a single value threaded into every suite. It works, but it couples unrelated concerns, prevents real test parallelism, and hides setup logic inside testify `BeforeTest`/`AfterTest` hooks ([test/e2e/tests/base/base_suite.go:367-389](test/e2e/tests/base/base_suite.go#L367-L389)).

A POC at [test/e2e/standalone_conformance/basicrouting_gateway_with_route/gateway_with_route_test.go](test/e2e/standalone_conformance/basicrouting_gateway_with_route/gateway_with_route_test.go) uses the gateway-api conformance suite directly. That validates the approach; now we need shared base packages so new tests can be written quickly and consistently.

Goal: design `test/conformance-e2e/kgwtest` (and sub-packages) that wrap gateway-api's `suite.ConformanceTestSuite` with kgateway-specific metadata, fixtures, and assertions, without forking upstream. Existing `test/e2e/features/*` tests migrate over time; the old framework stays in place during migration.

## Directory layout

New code lives under `test/conformance-e2e/`, clearly separated from the legacy `test/e2e/`. Framework and tests share the top-level directory:

```
test/
├── conformance-e2e/               # new code lives here
│   ├── kgwtest/                   # shared framework (imported as .../test/conformance-e2e/kgwtest)
│   │   ├── test.go                # kgwtest.Test struct + Run()
│   │   ├── suite.go               # kgwtest.NewSuite() + Options
│   │   ├── version.go             # GW API version parsing + gating
│   │   ├── setup.go               # VersionedSetup and suite-scoped fixture application
│   │   ├── namespace.go           # per-test namespace + manifest application
│   │   ├── diagnostics.go         # on-failure dump hook
│   │   ├── install/               # see "kgateway install" section
│   │   │   └── install.go
│   │   └── assertions/
│   │       ├── gateway/
│   │       ├── httproute/
│   │       ├── policy/
│   │       ├── envoy/
│   │       └── metrics/
│   ├── basic-routing/                          # one Go package per feature
│   │   ├── basic_routing.go                    # importable: ManifestFS, Tests, TestNamespace, Setup
│   │   ├── gateway_with_route.go               # importable: scenario + init() registration
│   │   ├── multiple_listeners.go               # importable: another scenario, same package
│   │   ├── suite_test.go                       # _test entry point: TestBasicRouting(t)
│   │   └── testdata/*.yaml                     # shared across scenarios in this feature
│   ├── cors/
│   │   └── ...
│   └── ...
└── e2e/                                        # existing homegrown framework + tests (unchanged during migration)
```

### Feature packages: importable scenarios + thin test entry

Each feature package has **two kinds of files**:

- **Importable files** (no `_test` suffix, build-tagged `//go:build e2e`): hold the scenarios as a slice of `kgwtest.Test`, the embedded `ManifestFS`, the shared `TestNamespace` constant, and the suite-level `Setup`. These are exported so the package can be reused — composed into a larger suite, extended with additional scenarios, or filtered.
- **Entry-point file** (`suite_test.go`): the only `_test.go` file in the package. Holds the `TestX(t)` function `go test` invokes. Constructs the `kgwtest.Suite` and runs `package.Tests`.

```go
// basic_routing.go (importable: package basicrouting)
//go:build e2e

package basicrouting

import (
    "embed"
    "io/fs"
    "github.com/kgateway-dev/kgateway/v2/test/conformance-e2e/kgwtest"
)

//go:embed testdata/*.yaml
var manifestsFS embed.FS

var ManifestFS fs.FS = manifestsFS

const TestNamespace = "kgw-e2e-basic-routing"

var Setup = kgwtest.VersionedSetup{
    Default: kgwtest.Setup{Manifests: []string{"testdata/_suite.yaml"}},
}

var Tests []kgwtest.Test
```

```go
// gateway_with_route.go (importable: package basicrouting)
//go:build e2e

package basicrouting

func init() { Tests = append(Tests, gatewayWithRouteTest) }

var gatewayWithRouteTest = kgwtest.Test{
    ShortName: "GatewayWithRoute",
    Manifests: []string{"testdata/gateway-with-route.yaml"},
    Parallel:  true,
    Test:      func(t *testing.T, ctx kgwtest.TestContext) { /* ... */ },
}
```

```go
// suite_test.go (entry point: package basicrouting_test)
//go:build e2e

package basicrouting_test

func TestBasicRouting(t *testing.T) {
    s := kgwtest.NewSuite(t, kgwtest.Options{
        GatewayClassName: "kgateway",
        ManifestFS:       []fs.FS{basicrouting.ManifestFS},
        VersionedSetup:   basicrouting.Setup,
    })
    require.NoError(t, s.Run(t, basicrouting.Tests))
}
```

Result: suite setup + fixtures apply once per `go test ./test/conformance-e2e/basic-routing/...`, and every scenario runs against that one suite. Scenarios with `Parallel: true` run concurrently against the shared fixtures. Adding a new scenario is a new importable `.go` file with an `init()` — no changes to `suite_test.go`.

## Reuse and extension

The framework is built around composition, not inheritance. There is no base suite to embed and no lifecycle hooks to override; instead, every reusable piece is a value an external caller can construct, mutate, and feed into `kgwtest.NewSuite`. This makes it easy for another module to:

- **Reuse the existing scenarios as-is**: import the feature package, hand `package.Tests` and `package.ManifestFS` to its own `kgwtest.NewSuite`. The entry-point `_test.go` is the only file unique to the upstream binary; everything the test bodies need is exported.
- **Add scenarios alongside the existing ones**: build a slice with `append(basicrouting.Tests, extraTests...)` and pass it to `Suite.Run`. Order is preserved, the suite-level fixture from `basicrouting.Setup` still applies, and the new scenarios share the existing `TestNamespace` (or set their own `Test.Namespace` for isolation).
- **Filter scenarios out**: walk `package.Tests` and drop entries by `ShortName` before passing to `Suite.Run`. Or use `Options.SkipTests`.
- **Replace the suite-level setup**: ignore `package.Setup` and pass a different `VersionedSetup`. The exported `TestNamespace` constant tells the new setup which namespace the test bodies expect.
- **Install a different control plane**: skip `kgwtest.Main` / pre-call `kgwtest/install.InstallCore` with different chart paths and values. The `kgwtest.Suite` itself is install-agnostic — it operates against whatever's already running.
- **Extend the client scheme**: pass `Options.Schemes []func(*runtime.Scheme) error` so non-default API groups (CRDs introduced by an extension) are decodable from the conformance suite's `client.Client`.
- **Wrap the entry point with extra setup**: a downstream entry function does its own pre-suite work (install extension, apply additional fixtures, register cleanup), then calls `kgwtest.NewSuite` and `Suite.Run` with whatever scenario slice it wants. There's no upstream hook to register against — just regular Go control flow before/after the suite call.

The compose-don't-inherit shape means a downstream that wants different behavior writes its own entry point rather than overriding lifecycle methods on a shared base. Everything the suite needs is a value passed into `Options`; everything the scenarios need is a value reachable from the imported package.

### Composing multiple feature packages

Because feature packages export their `Tests` slice and `ManifestFS`, an entry point can pull scenarios from several packages into one suite. Three common patterns:

**Pattern 1: union — one suite, scenarios from many packages.**

Hand the combined slice to a single `kgwtest.Suite`. All scenarios share the same suite-level fixtures (so the union of `Setup` manifests must be applied), the same `GatewayClassName`, and run as subtests under one parent `*testing.T`.

```go
func TestCombined(t *testing.T) {
    s := kgwtest.NewSuite(t, kgwtest.Options{
        GatewayClassName: "kgateway",
        ManifestFS:       []fs.FS{basicrouting.ManifestFS, cors.ManifestFS},
        VersionedSetup: kgwtest.VersionedSetup{
            Default: kgwtest.Setup{Manifests: []string{
                "testdata/basic-routing/_suite.yaml",
                "testdata/cors/_suite.yaml",
            }},
        },
    })
    tests := append(append([]kgwtest.Test{}, basicrouting.Tests...), cors.Tests...)
    require.NoError(t, s.Run(t, tests))
}
```

Right when the constituent packages already coexist namespace-wise (each uses its own `TestNamespace`) and downstream wants one CI job for them. The cost is that suite-level setup grows to the union, so any one package's setup failure fails the whole suite.

**Pattern 2: separate sub-suites — one binary, one suite per package.**

Each feature package gets its own `kgwtest.Suite` and runs as a subtest. Suite setups stay isolated, fixtures don't share namespaces by accident, and a setup failure in one package leaves the others runnable.

```go
func TestAllFeatures(t *testing.T) {
    t.Run("basic-routing", func(t *testing.T) {
        s := kgwtest.NewSuite(t, kgwtest.Options{
            GatewayClassName: "kgateway",
            ManifestFS:       []fs.FS{basicrouting.ManifestFS},
            VersionedSetup:   basicrouting.Setup,
        })
        require.NoError(t, s.Run(t, basicrouting.Tests))
    })
    t.Run("cors", func(t *testing.T) {
        s := kgwtest.NewSuite(t, kgwtest.Options{
            GatewayClassName: "kgateway",
            ManifestFS:       []fs.FS{cors.ManifestFS},
            VersionedSetup:   cors.Setup,
        })
        require.NoError(t, s.Run(t, cors.Tests))
    })
}
```

Right when packages have meaningfully different setups, different supported features, or you want `go test -run TestAllFeatures/cors` to filter at the package boundary.

**Pattern 3: filter + extend — base package + downstream additions.**

A downstream consumer takes the upstream `Tests` slice, drops scenarios that don't apply, and appends its own. The shared `ManifestFS` and `Setup` are reused as-is; downstream-specific manifests come from a second `fs.FS` layered into `Options.ManifestFS`.

```go
func TestDownstream(t *testing.T) {
    keep := slices.DeleteFunc(slices.Clone(basicrouting.Tests), func(tc kgwtest.Test) bool {
        return tc.ShortName == "GatewayWithRoute" // covered by a downstream variant
    })
    keep = append(keep, downstreamRoutingTests...)

    s := kgwtest.NewSuite(t, kgwtest.Options{
        GatewayClassName: "kgateway-extension",
        ManifestFS:       []fs.FS{basicrouting.ManifestFS, downstreamManifests},
        VersionedSetup:   basicrouting.Setup,
    })
    require.NoError(t, s.Run(t, keep))
}
```

Right when downstream wants the upstream coverage minus a few cases plus its own scenarios under one CI invocation.

## Core type: `kgwtest.Test`

A flat struct (no embedding) with the fields kgateway tests need:

```go
// test/conformance-e2e/kgwtest/test.go
type Test struct {
    ShortName       string
    Description     string
    Manifests       []string                   // applied verbatim before Test runs
    ManifestTransforms []ManifestTransform     // applied in order to each manifest's bytes
    Labels          []string                   // see "Labels" section
    Parallel, Slow, Provisional bool
    Features        []features.FeatureName

    MinGwApiVersion string                     // "1.2.0", empty = no lower bound
    MaxGwApiVersion string
    RequireChannel  Channel                    // ChannelStandard, ChannelExperimental, or ""

    // Namespace, if set, causes the framework to create a namespace with
    // this exact name before the test and delete it after. Manifests must
    // hardcode the same name. Empty (default) means the test uses shared
    // namespaces created by VersionedSetup.
    Namespace string

    Test func(t *testing.T, ctx TestContext)
}

type TestContext struct {
    Suite     *suite.ConformanceTestSuite
    Namespace string                           // empty when Test.Namespace is unset
}

// ManifestTransform mutates raw manifest bytes before they are parsed and
// applied. The Suite is provided so transforms can branch on detected
// Gateway API version/channel. A transform that has nothing to do should
// return its input unchanged so the underlying YAML stays kubectl-applyable.
type ManifestTransform func(s *Suite, in []byte) []byte
```

### Manifest transforms

Templating is off the table for the reasons in the parallelism section: manifests are kubectl-applyable as written. Manifest transforms are the escape hatch for cross-cutting compatibility shims that genuinely can't be encoded as static YAML — for example, rewriting a kind name when an API moves between channels. The transform inspects the bytes, decides whether the suite is in a context that needs the rewrite, and either returns the input unchanged or returns a modified copy:

```go
var test = kgwtest.Test{
    Manifests:          []string{"testdata/listenerset-policy.yaml"},
    ManifestTransforms: []kgwtest.ManifestTransform{kgwtest.TransformListenerSetForGwApiVersion},
}
```

`TransformListenerSetForGwApiVersion` ([kgwtest/transforms.go](test/conformance-e2e/kgwtest/transforms.go)) is the canonical example: on experimental channels below 1.5.1 it rewrites `ListenerSet` to the legacy `XListenerSet` form; on every other release it returns the input unchanged. New transforms live in the same file and follow the same pattern. The default for any author is to add nothing — most manifests don't need it.

The Test body receives a `TestContext` rather than the bare conformance suite — gives us a place to add per-test fields without breaking signatures.

## GW API version gating — per-test and per-suite

Suite level: added to `Options`. If unmet, `NewSuite` returns immediately and all tests skip with a clear reason. Useful for a whole test binary that only makes sense on 1.3+, or only on experimental channel.

```go
type Options struct {
    // ...
    MinGwApiVersion  string     // suite-wide lower bound
    MaxGwApiVersion  string     // suite-wide upper bound
    RequireChannel   string     // suite-wide channel requirement
    VersionedSetup   VersionedSetup
}
```

Per-test level: `MinGwApiVersion`, `MaxGwApiVersion`, `RequireChannel` on `kgwtest.Test`. Checked by `shouldSkip()` before namespace creation.

Detection mirrors [gateway-api/conformance/utils/suite/suite.go:694-723](../kubernetes-sigs/gateway-api/conformance/utils/suite/suite.go#L694-L723) — read CRD annotations `gateway.networking.k8s.io/bundle-version` and `channel` at suite init.

Version comparison uses `golang.org/x/mod/semver` (add `v` prefix before comparing).

## Labels

Tests carry an optional `Labels []string` for free-form classification — `smoke`, `slow`, `extension-only`, a feature name, a stability bucket, anything an author or CI job wants to filter on. The framework gives every test the same neutral mechanism rather than a fixed taxonomy.

Two suite-level filters consume them:

- `Options.RunLabels []string` — if non-empty, restricts the suite to tests with at least one matching label. Defaults from `KGWTEST_RUN_LABELS` (comma-separated).
- `Options.SkipLabels []string` — skips any test that has at least one matching label. Defaults from `KGWTEST_SKIP_LABELS`.

Match semantics are "any-of": a test passes the run filter if its label set overlaps the run filter, and is dropped by the skip filter if its label set overlaps the skip filter. The two filters are independent and combine with `RunTest` / `SkipTests` (which match on `ShortName`).

Labels are off by default. Adding labels to a test should be motivated by an actual filter someone wants to run; speculative labels are noise. CI jobs that need a fast pre-merge subset can add `Labels: []string{"smoke"}` to the relevant scenarios and run with `KGWTEST_RUN_LABELS=smoke`.

## Versioned setup

Port the existing pattern from [test/e2e/tests/base/base_suite.go:197,296-336](test/e2e/tests/base/base_suite.go#L197) into the new framework with the same selection semantics (highest matching version `<=` current, within matching channel, else Default):

```go
// test/kgwtest/setup.go
type VersionedSetup struct {
    Default   Setup                            // applied if no ByVersion match
    ByVersion map[Channel]map[string]Setup     // Channel -> semver -> Setup (semver without "v" prefix, matching existing style)
}

type Setup struct {
    Manifests []string                         // paths resolved via Suite.ManifestFS
    // Extension point: post-apply hook, e.g. waiting for a Deployment not in the manifests
    PostApply func(t *testing.T, s *Suite)
}

type Channel string

const (
    ChannelStandard     Channel = "standard"
    ChannelExperimental Channel = "experimental"
)
```

At `NewSuite`, after version detection, `selectSetup(VersionedSetup)` picks the active `Setup` and applies it once. Cleanup registered against the parent `*testing.T`.

Existing callers like [test/e2e/features/metrics/suite.go:36-42](test/e2e/features/metrics/suite.go#L36-L42) map cleanly onto this shape when migrated.

## Parallelism and namespaces

Upstream `ConformanceTest.Run` already calls `t.Parallel()` when `Parallel: true` is set ([gateway-api/conformance/utils/suite/conformance.go:42-45](../kubernetes-sigs/gateway-api/conformance/utils/suite/conformance.go#L42-L45)) — we preserve that.

**No templating.** Manifests in `testdata/` are applied verbatim. They reference namespaces by literal name and hardcode `gatewayClassName: kgateway` (or `kgateway-waypoint` etc.), so `kubectl apply -f testdata/...` works directly during local debugging — the same YAML the test framework runs.

The model is "shared namespaces by default, opt-in isolation":

1. **Shared namespaces** (the default). Each feature package declares one or more namespaces in a `_suite.yaml` manifest applied via `VersionedSetup.Default.Manifests`. Every scenario in the package puts its resources into those namespaces with names unique per scenario (e.g., `gateway-with-route-svc` vs `multiple-listeners-svc`). This matches upstream gateway-api conformance, which uses fixed namespaces like `gateway-conformance-infra` for the same reason.
2. **Per-test namespaces** (opt-in). A scenario that needs isolation — e.g., it watches for a resource to be deleted, or installs cluster-scoped resources — sets `Test.Namespace = "kgw-e2e-<unique-name>"`. The framework creates the namespace before the test, deletes it after, and exposes the name as `ctx.Namespace` in the test body. Manifests for that scenario hardcode the same namespace name. `kgwtest.Suite.Run` validates that no two Tests share a non-empty `Namespace` value.
3. **Why parallel-safe**: parallel scenarios in the same shared namespace must use unique resource names. Parallel scenarios that opt into isolation get fully separate namespaces. Both modes work together.

`gatewayClassName` is **not** rewritten. Manifests hardcode the class they need. `Options.GatewayClassName` stays — it tells the upstream conformance suite which class to look up for ControllerName resolution and feature discovery — but it does not mutate applied manifests. Suites can mix kgateway-managed classes (`kgateway`, `kgateway-waypoint`, etc.) freely, since they all map to the same ControllerName.

## Assertions

Free functions grouped by domain, no shared state, no receivers. Each function takes `(t, client, timeoutConfig, identifiers..., matcher)` — the last parameter is a struct (or predicate) describing what to check. One function per *verb*, not one function per field.

### Polling and "eventually" — reuse upstream primitives

Upstream uses two mechanisms; we reuse both and don't introduce a third:

- **Status waits** (wait for a resource's condition / status to reach a desired state): `k8s.io/apimachinery/pkg/util/wait.PollUntilContextTimeout`, driven by timeouts from `TimeoutConfig` ([gateway-api/conformance/utils/kubernetes/helpers.go](../kubernetes-sigs/gateway-api/conformance/utils/kubernetes/helpers.go)).
- **HTTP stability waits** (wait for the data plane to consistently return the expected response): `http.AwaitConvergence`, which requires `TimeoutConfig.RequiredConsecutiveSuccesses` consecutive passes; any failure resets the counter ([gateway-api/conformance/utils/http/http.go:276-292](../kubernetes-sigs/gateway-api/conformance/utils/http/http.go#L276-L292)).

No Gomega `Eventually`, no custom retry loops. Our kgateway assertions take `config.TimeoutConfig` from the suite and use these primitives directly:

```go
func MustHaveCondition(t *testing.T, c client.Client, tc config.TimeoutConfig,
    nn types.NamespacedName, match ConditionMatch) {
    t.Helper()
    err := wait.PollUntilContextTimeout(ctx, tc.DefaultPollInterval, tc.SomeStatusTimeout, true,
        func(ctx context.Context) (bool, error) {
            // fetch resource, evaluate match, return (done, error)
        })
    require.NoError(t, err, "...")
}
```

For anything that needs to be *stable* (not just true once — e.g., metrics values, Envoy config reflecting a policy), use `http.AwaitConvergence` or a small wrapper that gives it non-HTTP semantics.

### HTTP response assertions — reuse upstream

Don't invent anything. `httputils.ExpectedResponse` already accepts a rich, declarative matcher:

```go
httputils.MakeRequestAndExpectEventuallyConsistentResponse(t, rt, tc, gwAddr, httputils.ExpectedResponse{
    Request: httputils.Request{
        Host: "example.com",
        Path: "/api",
        Headers: map[string]string{"x-test": "1"},
    },
    Response: httputils.Response{
        StatusCodes: []int{200},
        Headers:     map[string]string{"x-upstream": "echo"},
        AbsentHeaders: []string{"x-stripped"},
    },
    Backend:   "example-svc",
    Namespace: "example",
})
```

Header checks, status-code assertions, absent-headers, body matching — all covered by upstream fields. Do not wrap this in kgateway-specific helpers unless we need a field upstream doesn't provide.

### Resource status assertions — matcher structs

For kgateway CRD status checks, take a matcher struct describing the condition(s) to wait for. One assertion function per *shape of wait*, not per field:

```go
// test/conformance-e2e/kgwtest/assertions/policy/traffic_policy.go
package policy

type ConditionMatch struct {
    Type       string            // required
    Status     metav1.ConditionStatus   // required
    Reason     string            // optional — exact match if set
    ReasonIn   []string          // optional — any-of match
    MessageContains string       // optional — substring
}

func MustHaveCondition(t *testing.T, c client.Client, tc config.TimeoutConfig,
    nn types.NamespacedName, match ConditionMatch) { ... }

func MustHaveAncestor(t *testing.T, c client.Client, tc config.TimeoutConfig,
    nn types.NamespacedName, ancestor gwv1.ParentReference, match ConditionMatch) { ... }
```

One `MustHaveCondition` covers every reason/status/message combination via the struct, instead of `MustBeAccepted` / `MustBeRejected` / `MustHaveReason_X` / etc. Convenience wrappers (`MustBeAccepted = MustHaveCondition{Type:"Accepted", Status:True}`) can be added later only if call sites actually benefit from them.

Field convention follows upstream `ExpectedResponse`: **zero-value / nil = don't check**. `ConditionMatch{Type:"Accepted", Status:True}` matches any reason and any message; setting `Reason` or `MessageContains` tightens the match. Consistent with how `httputils.ExpectedResponse` handles `Protocol == ""` and `StatusCodes == nil` ([gateway-api/conformance/utils/http/http.go:317-506](../kubernetes-sigs/gateway-api/conformance/utils/http/http.go#L317-L506)). Assertions use `testify/require` for reporting.

### Envoy config-dump / metrics — matcher structs or predicates

Same pattern. A single function per thing-we-check, parametrized by a matcher:

```go
// envoy/clusters.go
type ClusterMatch struct {
    Name          string        // exact, optional
    NameContains  string        // substring, optional
    LbPolicy      string        // optional
    EndpointCount *int          // optional — nil = don't check
}
func MustHaveCluster(t *testing.T, pod types.NamespacedName, match ClusterMatch) { ... }

// metrics/metrics.go — when struct-shaped matchers get unwieldy, accept a predicate
func MustObserve(t *testing.T, pod types.NamespacedName, metric string,
    labels map[string]string, predicate func(value float64) bool) { ... }
```

Rule of thumb:
- Small, stable set of fields to check -> matcher struct with optional fields (nil/zero = don't care).
- Open-ended check over a numeric or computed value -> predicate `func(T) bool` (or an error-returning one for richer reporting).
- Never one function per field.

### Packages

- `kgwtest/assertions/gateway` — kgateway-specific Gateway wait helpers (reuse `gwconformance.GatewayAndHTTPRoutesMustBeAccepted` where it fits)
- `kgwtest/assertions/httproute` — kgateway-specific route checks
- `kgwtest/assertions/policy` — TrafficPolicy, BackendPolicy, HTTPListenerPolicy status
- `kgwtest/assertions/envoy` — config-dump (clusters, listeners, routes), cluster presence
- `kgwtest/assertions/metrics` — controller metrics scrape + predicate matching

No suite/TestInstallation handle — inputs are always `(t, client, timeoutConfig, identifiers..., matcher)`. Mirrors gateway-api's [kubernetes/helpers.go](../kubernetes-sigs/gateway-api/conformance/utils/kubernetes/helpers.go) and envoy-gateway's [test/e2e/tests/utils.go](../envoyproxy/gateway/test/e2e/tests/utils.go), but with matcher structs instead of per-field functions.

## Suite construction

```go
// test/kgwtest/suite.go
type Suite struct {
    Conformance *suite.ConformanceTestSuite
    apiVersion  string
    apiChannel  Channel
    activeSetup Setup                      // the Setup selected from VersionedSetup
    opts        Options
}

type Options struct {
    GatewayClassName  string
    ManifestFS        []fs.FS
    SupportedFeatures sets.Set[features.FeatureName]

    // Schemes are applied to the conformance suite's runtime.Scheme so
    // CRDs from extension API groups (anything beyond gateway.networking.k8s.io
    // and the kgateway core groups) are decodable through Suite.Client.
    // Each function receives the scheme and adds its types via the standard
    // AddToScheme pattern.
    Schemes []func(*runtime.Scheme) error

    VersionedSetup    VersionedSetup       // empty is valid — no suite fixtures
    MinGwApiVersion   string
    MaxGwApiVersion   string
    RequireChannel    string

    RunTest           string               // filter to single ShortName; reads KGWTEST_RUN_TEST
    SkipTests         []string             // reads KGWTEST_SKIP_TESTS
    RunLabels         []string             // reads KGWTEST_RUN_LABELS
    SkipLabels        []string             // reads KGWTEST_SKIP_LABELS
}

func NewSuite(t *testing.T, opts Options) *Suite
func (s *Suite) Run(t *testing.T, tests []Test) error
```

`Options` fields default from env vars (`KGWTEST_RUN_TEST`, `KGWTEST_SKIP_TESTS`) when not set programmatically.

## Failure diagnostics

Wire `Hook` on the underlying `suite.ConformanceTestSuite` to `diagnostics.Dump` ([test/kgwtest/diagnostics.go]): on test failure, collect kgateway controller logs, Envoy config-dump from the Gateway pod, `kubectl get -o yaml` of all resources in the per-test namespace. Write to `$ARTIFACTS_DIR/<ShortName>/`. Same pattern envoy-gateway uses in [test/e2e/tests/utils.go](../envoyproxy/gateway/test/e2e/tests/utils.go) `CollectAndDump`.

## kgateway install — how it fits in

**Current state** (important to understand before choosing):

- The existing homegrown framework installs kgateway **from inside the test binary** via `TestInstallation.InstallKgatewayFromLocalChart()` ([test/e2e/tests/kgateway_test.go:52](test/e2e/tests/kgateway_test.go#L52)). It reads `install.Context` (namespace, values files, extra helm args) off `TestInstallation.Metadata` and shells out to helm via `TestInstallation.Actions`.
- CI runs `make e2e-test` which is just `go test -tags e2e ./...` ([Makefile:215-218](Makefile#L215-L218)). No external install step — the binary does it.
- The conformance POC ([test/e2e/standalone_conformance/basicrouting_gateway_with_route/gateway_with_route_test.go](test/e2e/standalone_conformance/basicrouting_gateway_with_route/gateway_with_route_test.go)) **does not install kgateway** — it assumes it's already running. Works during dev (Tilt provides it); would fail in CI today without a new install step.

**Plan**: keep install **separable from the test framework**, but provide it as an opt-in helper so tests can still "just run" with `go test`.

1. **`test/kgwtest/install`** (new package) — thin, focused wrapper around `helm install/upgrade`. Exports:
   ```go
   type Config struct {
       Namespace        string
       ReleaseName      string                 // default "kgateway"
       ChartPath        string                 // local path (default) or OCI ref
       ValuesFiles      []string
       ExtraArgs        []string
       WaitTimeout      time.Duration
   }
   func Install(ctx context.Context, t *testing.T, cfg Config) error
   func Uninstall(ctx context.Context, t *testing.T, cfg Config) error
   ```
   Replaces `TestInstallation.InstallKgatewayFromLocalChart` for new-style tests. Sole job is running helm — no other responsibilities.

2. **`kgwtest.Suite` does NOT install kgateway itself.** Test binaries that need to install call `install.Install` from their own `TestMain` (or before `NewSuite`). Binaries that run against an existing install (dev loop with Tilt, shared CI cluster) skip it. This keeps `kgwtest` focused on running tests.

3. **Standard CI pattern** for a new-style test binary — wrapped in a one-line helper so we don't paste 15 lines of boilerplate into every binary:

   ```go
   // test/conformance-e2e/kgwtest/main.go
   package kgwtest

   type MainOptions struct {
       // Install, if non-nil, installs kgateway core via helm before m.Run()
       // and uninstalls after. Skipped when KGW_SKIP_INSTALL is set.
       Install *install.CoreConfig

       // InstallCRDs, if non-nil, also installs the CRDs chart. Sequenced
       // before Install. Skipped when KGW_SKIP_INSTALL is set.
       InstallCRDs *install.CRDsConfig
   }

   // Main is a TestMain helper for the common case: install kgateway, run
   // tests, uninstall. Reads KGW_SKIP_INSTALL / KGW_CHART_PATH / KGW_VALUES
   // for defaults so dev (tilt up + KGW_SKIP_INSTALL=1) and CI (default
   // helm install) share the same binary.
   func Main(m *testing.M, opts MainOptions)

   // MainWithDefaults pulls chart paths, namespace, values from env vars and
   // installs kgateway. Suitable for almost every binary.
   func MainWithDefaults(m *testing.M)
   ```

   Test binary in the common case:

   ```go
   func TestMain(m *testing.M) { kgwtest.MainWithDefaults(m) }
   ```

   With overrides (e.g., per-binary values file):

   ```go
   func TestMain(m *testing.M) {
       kgwtest.Main(m, kgwtest.MainOptions{
           Install: &install.CoreConfig{ValuesFiles: []string{"testdata/values.yaml"}},
       })
   }
   ```

   Dev loop sets `KGW_SKIP_INSTALL=1` and uses Tilt. CI leaves it unset. The helper exists to remove duplication, not to hide the path — `install.InstallCore` / `install.UninstallCore` remain public for binaries that want imperative control.

4. **When NOT to use `kgwtest.Main`**: tests that exercise the install flow itself, or need multiple releases, or need install-and-restart cycles. Those write their own `TestMain` (or do install in the test entry function) and call `install.*` directly. `multi-install` is the canonical example — it installs CRDs once and core twice in different namespaces with different values, none of which fits a single-config helper.

5. **Per-binary install variations**: supported naturally — each test binary calls `install.InstallCore` with whatever values files it needs. If two binaries want different kgateway configurations, they run in different CI jobs. We don't need per-test install; that was never really used in the legacy framework either.

6. **`TestInstallation.Metadata` / `Actions` do not get ported.** `Metadata` becomes `install.Config` arguments. `Actions` (kubectl wrapper) is replaced by direct use of `controller-runtime/client` (already on the conformance suite as `suite.Client`) plus the applier. `Metadata.Actions.Helm` becomes a small internal helper inside `kgwtest/install`.

## Migration

- New framework lives at `test/conformance-e2e/kgwtest/` and new tests at `test/conformance-e2e/<feature>/` from day one. No churn in `test/e2e/tests/` or `test/e2e/features/`.
- Migrate the existing POC ([test/e2e/standalone_conformance/basicrouting_gateway_with_route/](test/e2e/standalone_conformance/basicrouting_gateway_with_route/)) to `test/conformance-e2e/basic-routing/` using `kgwtest`. Validates the framework end-to-end.
- CI: extend the e2e matrix in [.github/workflows/e2e.yaml](.github/workflows/e2e.yaml) with a job for `test/conformance-e2e/...`. The job runs a setup step (`make kind-build-and-load` then a helm install — or calls a new `make install-kgateway-for-conformance` target) before `go test`.
- As features are migrated from `test/e2e/features/*`, delete the old suite in the same PR. `BaseTestingSuite` stays until the last caller is gone; only then does it get removed.
- The two frameworks do not share code. No interop.

## Critical files

- `test/conformance-e2e/kgwtest/test.go` — new
- `test/conformance-e2e/kgwtest/suite.go` — new
- `test/conformance-e2e/kgwtest/version.go` — new
- `test/conformance-e2e/kgwtest/setup.go` — new (VersionedSetup selection)
- `test/conformance-e2e/kgwtest/namespace.go` — new
- `test/conformance-e2e/kgwtest/diagnostics.go` — new
- `test/conformance-e2e/kgwtest/install/install.go` — new (helm wrapper)
- `test/conformance-e2e/kgwtest/assertions/{gateway,httproute,policy,envoy,metrics}/*.go` — new
- `test/conformance-e2e/basic-routing/gateway_with_route_test.go` + `testdata/*.yaml` — move + rewrite from the existing POC to use `kgwtest`
- `.github/workflows/e2e.yaml` — add job for `test/conformance-e2e/...`
- `Makefile` — add `install-kgateway-for-conformance` target (thin wrapper over helm, same args the `install.Install` helper uses)

## Verification

1. Convert the existing POC to use `kgwtest.NewSuite` and `kgwtest.Test`. Must pass against a local kind cluster (`make kind-build-and-load` + helm install, or `tilt up` + `KGW_SKIP_INSTALL=1`).
2. Add a second trivial test in a new directory with `Parallel: true`. Verify both tests run in parallel against the same suite-scoped fixtures (log timestamps overlap).
3. Add a test with `MinGwApiVersion: "99.0.0"` and verify it skips with a clear reason that names the detected vs required version.
4. Add a suite with `Options.MinGwApiVersion: "99.0.0"` and verify the whole suite skips cleanly.
5. Configure a `VersionedSetup` with `Default` and a `ByVersion[ChannelStandard]["1.5.0"]` entry. Run against a cluster with GW API 1.4 (expect Default) and 1.5.1 (expect the versioned entry).
6. Force a failure in a test and confirm `$ARTIFACTS_DIR/<ShortName>/` contains controller logs, Envoy config-dump, and resource YAMLs from the per-test namespace.
7. Unit test `selectSetup`, version comparison, and `shouldSkip` with a table-driven test covering version bounds, channel mismatch, and `RunTest`/`SkipTests` overrides.
8. Run `make analyze` and `make verify` — no regressions.

## Phase 2: extensions for complex existing suites

The Phase 1 framework (already implemented) covers suites like `basic-routing` whose pattern is *apply some Gateway/HTTPRoute manifests, send a request, check the response*. Three more demanding suites — `multiinstall`, `deployer`, `zero_downtime_rollout` — exercise capabilities Phase 1 does not yet provide. This section catalogs the gaps and the specific packages that close them.

### Per-suite gap analysis

**multiinstall** ([test/e2e/features/multiinstall/](test/e2e/features/multiinstall/), orchestrated by [test/e2e/tests/multiple_installs_test.go](test/e2e/tests/multiple_installs_test.go))

- Validates two parallel kgateway helm releases in different namespaces with different `discoveryNamespaceSelectors` values, asserting each only manages its own namespace.
- **Needs**: helm install (CRDs once + core twice with different values files) at the test entry point; in-cluster curl from a fixture pod into the gateway's ClusterIP service; access-log assertions via container logs.
- **Does NOT need**: per-test namespaces created by the framework — the install namespaces ARE the test namespaces. Override `Test.Namespace` (or skip framework-managed namespaces entirely for this one suite).

**deployer** ([test/e2e/features/deployer/suite.go](test/e2e/features/deployer/suite.go))

- Validates kgateway's deployer component: when a `Gateway` resource is created, the deployer provisions Deployment/Service/ServiceAccount, and `GatewayParameters` controls Envoy config, replicas, security context, HPA/PDB/VPA. Eight tests with eight different manifest sets.
- **Needs**: per-test manifests (already supported via `Test.Manifests`); typed-client status assertions on Deployment/HPA/PDB/VPA; client-runtime patch helpers (Gateway / GatewayParameters); Envoy admin API access via port-forward; "must not exist" assertions (self-managed Gateway must NOT create proxy resources); a non-standard CRD (VPA) installed at suite setup.
- **Does NOT need**: helm install during the test. Pre-installed kgateway is fine.
- **Per-test namespaces**: confirmed direction — each deployer test runs in its own `kgw-e2e-<shortname>` namespace. Verify during migration that the deployer correctly provisions proxy resources in arbitrary namespaces; if not, that's a deployer bug to fix, not a framework workaround.

**zero_downtime_rollout** ([test/e2e/features/zero_downtime_rollout/suite.go](test/e2e/features/zero_downtime_rollout/suite.go))

- Spawns `hey` load generator from a fixture pod via `kubectl exec`, restarts the gateway proxy Deployment twice during the load run, and asserts zero non-200 responses.
- **Needs**: `kubectl exec` (background, non-blocking, with stdout capture); `kubectl rollout restart deployment` with wait; `EventuallyPodsRunning(label, namespace)` helper.
- **Does NOT need**: helm install. The `hey` and gateway pods are test-scoped fixtures.

### New packages

#### `kgwtest/cluster` — kubectl operations

A thin, focused wrapper around kubectl operations. **Operations only — no assertions live here.** Replaces `TestInstallation.Actions.Kubectl()` for new-style tests. Implementation can shell out to `kubectl` or use clientcmd + remotecommand directly; preference for the latter to avoid binary dependency, except where shelling out is materially simpler (e.g., `kubectl rollout status`).

```go
// test/conformance-e2e/kgwtest/cluster/cluster.go
package cluster

type Client struct {
    K8s     client.Client          // controller-runtime
    REST    *rest.Config
    Kubectl Kubectl                // shell wrapper for ops without good Go APIs
}

func New(c client.Client, rc *rest.Config) *Client

// kubectl exec, returns stdout/stderr; non-blocking variant for background load
func (c *Client) Exec(ctx context.Context, podNN types.NamespacedName, container string, cmd ...string) ([]byte, []byte, error)
func (c *Client) ExecBackground(ctx context.Context, podNN types.NamespacedName, container string, cmd ...string) (*BackgroundCmd, error)

// rollout restart deployment + wait for rollout status
func (c *Client) RestartDeployment(ctx context.Context, nn types.NamespacedName) error
func (c *Client) RestartDeploymentAndWait(ctx context.Context, nn types.NamespacedName, timeout time.Duration) error

// pods/logs
func (c *Client) PodsWithLabel(ctx context.Context, ns, selector string) ([]corev1.Pod, error)
func (c *Client) ContainerLogs(ctx context.Context, podNN types.NamespacedName, container string) (string, error)

// generic patch via merge
func (c *Client) Patch(ctx context.Context, obj client.Object, mutate func(client.Object)) error

// port-forward; returns local port and a cancel func
func (c *Client) PortForward(ctx context.Context, podNN types.NamespacedName, remotePort int) (localPort int, cancel func(), err error)
```

The `Suite` exposes `*cluster.Client` as `Suite.Cluster()` for tests that need it.

#### `kgwtest/install` (already in plan, refined)

Split into two operations matching the existing helm chart split:

```go
// test/conformance-e2e/kgwtest/install/install.go

type CRDsConfig struct {
    Namespace   string                 // helm release namespace (CRDs are cluster-scoped but the release lives somewhere)
    ChartPath   string                 // local path to crd chart
    ReleaseName string                 // default "kgateway-crds"
    WaitTimeout time.Duration
}

type CoreConfig struct {
    Namespace   string                 // install namespace
    ChartPath   string
    ReleaseName string                 // default "kgateway"
    ValuesFiles []string
    ExtraArgs   []string
    WaitTimeout time.Duration
}

func InstallCRDs(ctx context.Context, t *testing.T, cfg CRDsConfig) error
func UninstallCRDs(ctx context.Context, t *testing.T, cfg CRDsConfig) error
func InstallCore(ctx context.Context, t *testing.T, cfg CoreConfig) error
func UninstallCore(ctx context.Context, t *testing.T, cfg CoreConfig) error
```

**No `InstallReleases` field on `kgwtest.Options`** (decided). Test entry points call `install.*` directly. Multi-install does:

```go
func TestMultiInstall(t *testing.T) {
    ctx := context.Background()
    require.NoError(t, install.InstallCRDs(ctx, t, install.CRDsConfig{...}))
    t.Cleanup(func() { _ = install.UninstallCRDs(ctx, t, ...) })
    for _, ns := range []string{"kgw-test-1", "kgw-test-2"} {
        cfg := install.CoreConfig{Namespace: ns, ValuesFiles: []string{fmt.Sprintf("testdata/values-%s.yaml", ns)}}
        require.NoError(t, install.InstallCore(ctx, t, cfg))
        t.Cleanup(func() { _ = install.UninstallCore(ctx, t, cfg) })
    }
    // suite operates against both installs; tests iterate over namespaces
    s := kgwtest.NewSuite(t, kgwtest.Options{...})
    require.NoError(t, s.Run(t, tests))
}
```

#### `kgwtest/assertions/k8s` — typed status and existence

```go
// status.go
func MustHaveCondition(t *testing.T, c client.Client, tc config.TimeoutConfig, obj client.Object, match ConditionMatch)

// existence.go
func MustEventuallyExist(t *testing.T, c client.Client, tc config.TimeoutConfig, objs ...client.Object)
func MustEventuallyNotExist(t *testing.T, c client.Client, tc config.TimeoutConfig, objs ...client.Object)
func MustConsistentlyNotExist(t *testing.T, c client.Client, tc config.TimeoutConfig, dur time.Duration, objs ...client.Object)

// deployment.go
func MustHaveReadyReplicas(t *testing.T, c client.Client, tc config.TimeoutConfig, nn types.NamespacedName, count int)
func MustEventuallyPodsRunning(t *testing.T, c client.Client, tc config.TimeoutConfig, ns, selector string)
```

`ConditionMatch` is the same struct described in the Phase 1 assertions section. `MustHaveCondition` is generic — works for Gateway, HTTPRoute, Policy, anything with a typed condition list — by extracting via reflection or a small `WithConditions(client.Object) []metav1.Condition` accessor.

#### `kgwtest/assertions/curl` — in-cluster curl

```go
// test/conformance-e2e/kgwtest/assertions/curl/curl.go
type Pod struct {
    NN        types.NamespacedName
    Container string                 // default ""
}

type Target struct {
    Host string
    Port int
    Path string
    Headers map[string]string
}

// Sends curl from the given pod via kubectl exec; retries until response matches
// expected and stays consistent for tc.RequiredConsecutiveSuccesses runs.
func MustEventuallyMatch(t *testing.T, cli *cluster.Client, tc config.TimeoutConfig,
    pod Pod, target Target, expected httputils.ExpectedResponse)
```

Reuses upstream's `httputils.ExpectedResponse` matcher — no parallel definitions.

#### `kgwtest/assertions/envoy` — admin API

```go
// test/conformance-e2e/kgwtest/assertions/envoy/admin.go

// Opens a port-forward to the proxy pod's admin port, runs fn against the
// resulting localhost URL, returns when fn returns nil or after tc.MaxTimeToConsistency.
func MustReachAdmin(t *testing.T, cli *cluster.Client, tc config.TimeoutConfig,
    podNN types.NamespacedName, fn func(adminURL string) error)

// Convenience wrappers built on top:
type ClusterMatch struct { Name, NameContains, LbPolicy string; EndpointCount *int }
func MustHaveCluster(t *testing.T, cli *cluster.Client, tc config.TimeoutConfig,
    podNN types.NamespacedName, match ClusterMatch)

type LogLevelMatch struct { Default, Component string }
func MustHaveLogLevel(t *testing.T, cli *cluster.Client, tc config.TimeoutConfig,
    podNN types.NamespacedName, match LogLevelMatch)
```

Mirrors the kinds of admin queries the deployer suite makes today via `serverInfoLogLevelAssertion` and `xdsClusterAssertion`.

### Suite extensions

- **Scheme registration**: `kgwtest.Options.Schemes []func(*runtime.Scheme) error` — Phase 1 hardcodes the gateway-api schemes via `conformance.DefaultOptions`. Deployer needs `autoscalingv2`, `policyv1`. Make this opt-in via Options.
- **`Suite.Cluster() *cluster.Client`** — exposes the cluster operations wrapper. Lazily created.
- **Failure-dump diagnostics** ([kgwtest/diagnostics.go](test/conformance-e2e/kgwtest/diagnostics.go)) — currently a stub. Implement to dump kgateway controller logs, Envoy config-dump (via `cluster.PortForward`), and resource YAMLs from the per-test namespace. Reuses `cluster.Client`.

### Migration mapping

| Test suite | Per-test ns? | Suite setup | New packages used |
|---|---|---|---|
| `multi-install` | No (uses install ns) | `install.InstallCRDs` + `install.InstallCore` x2 in TestMain | `install`, `cluster`, `assertions/curl`, `assertions/k8s` |
| `deployer` | Yes (parallelizes) | `VersionedSetup` applies VPA CRD; opts in `autoscalingv2` + `policyv1` schemes | `cluster` (patch, port-forward), `assertions/k8s`, `assertions/envoy` |
| `zero_downtime_rollout` | Yes | Manifests apply gateway, backend, hey pod | `cluster` (Exec, ExecBackground, RestartDeploymentAndWait), `assertions/k8s` |

### What does NOT need to change

- Test struct, namespace templating, version gating, VersionedSetup, polling primitives — all unchanged. Phase 2 is purely additive packages.
- The `Test.Namespace` override (already on the struct) covers multi-install's case where the install namespace is the test namespace.

### Critical files (Phase 2)

- `test/conformance-e2e/kgwtest/cluster/cluster.go` — new
- `test/conformance-e2e/kgwtest/install/install.go` — new (CRDs + Core)
- `test/conformance-e2e/kgwtest/assertions/k8s/{status,existence,deployment}.go` — new
- `test/conformance-e2e/kgwtest/assertions/curl/curl.go` — new
- `test/conformance-e2e/kgwtest/assertions/envoy/admin.go` — new
- `test/conformance-e2e/kgwtest/suite.go` — add `Schemes []func(*runtime.Scheme) error` Option, add `Cluster()` accessor
- `test/conformance-e2e/kgwtest/diagnostics.go` — implement real failure dump using `cluster.Client`

### Phase 2 verification

1. Migrate `zero_downtime_rollout` first — smallest extension surface (`cluster.Exec` family + `MustEventuallyPodsRunning`). Must pass against kind with kgateway pre-installed.
2. Migrate `deployer` — exercises every assertion package and per-test namespace parallelism. Verify all 8 deployer test cases pass with `Parallel: true` (deployer-side bug if any flake).
3. Migrate `multi-install` — exercises `install.*` and in-cluster curl. Run with the helm chart paths used by `make e2e-test`.
4. Compare wall-clock time: legacy serial runs vs new parallel runs for `deployer`. Should be materially faster if parallel actually works.
5. Force a deployer test to fail; confirm the diagnostics hook dumps controller logs, Envoy admin output, and per-test-namespace resource YAMLs.

## Open Questions

These are design choices the framework leaves to team consensus rather than enforcing. Each captures the proposed default and the tradeoffs so the discussion has a starting point.

### File organization within a feature package: one file per scenario?

Proposal: **one `.go` file per scenario** inside a feature package. A "scenario" is a `kgwtest.Test` value with its own `Manifests` list and `ShortName` (e.g., `gateway_with_route.go`, `multiple_listeners.go` inside `basic-routing/`). Sub-cases that share a single Test body — variants of the same scenario tested via `t.Run` blocks against the same applied manifests — stay in the same file.

Reasons to default to one-file-per-scenario:

1. **Mirrors upstream gateway-api conformance.** Their [conformance/tests/](../kubernetes-sigs/gateway-api/conformance/tests/) directory is one `.go` file per scenario. Matching that layout makes the framework feel familiar to anyone coming from upstream.
2. **Greppable identity.** A `ShortName` maps directly to a filename, so failures in CI logs lead to the file they describe without an index lookup.
3. **Clear test-vs-subtest split rule.** Different manifest set or different setup -> separate `Test` value -> separate file. Same manifests, multiple variants of the request/assertion -> subtests inside one Test body. The rule is mechanical, not judgment-based.
4. **Merge conflict isolation.** Adding or modifying a scenario touches one file; concurrent PRs adding scenarios don't collide.
5. **Shared-state has a home.** The importable `basic_routing.go` (or whatever the package is named) holds `ManifestFS`, `TestNamespace`, `Setup`, and the `Tests` slice — the cross-cutting bits — while each scenario file holds only its own `kgwtest.Test`. Authors don't have to decide where the shared state goes.

When a feature package outgrows the one-file-per-scenario pattern (~15+ scenarios and discovery starts to suffer), the escape hatch is **subpackages**, not file consolidation: `basic-routing/routes/`, `basic-routing/listeners/`, each with its own `Tests` slice, composed at the parent via `append(routes.Tests, listeners.Tests...)`. That preserves the one-file-per-scenario rule inside each subpackage and keeps the parent entry point trivial.

Open for team discussion: the threshold at which subpackages should kick in, whether scenario files should also carry their `init()` registration (current proposal) or whether registration should be centralized in the importable package file. The first is more local; the second is easier to audit.

