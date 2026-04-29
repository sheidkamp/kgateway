//go:build e2e

// Package basicrouting holds the basic-routing conformance scenarios as a
// reusable []kgwtest.Test slice along with the embedded testdata. The
// scenarios are exported so other packages can build a suite over them — for
// example, by appending downstream-specific scenarios or wrapping them in a
// suite with extra setup.
package basicrouting

import (
	"embed"
	"io/fs"

	"github.com/kgateway-dev/kgateway/v2/test/conformance-e2e/kgwtest"
)

//go:embed testdata/*.yaml
var manifestsFS embed.FS

// ManifestFS is the embedded testdata for this package. Pass to
// kgwtest.Options.ManifestFS so the suite can resolve scenario manifest
// paths.
var ManifestFS fs.FS = manifestsFS

// TestNamespace is the shared namespace every scenario in this package puts
// its resources into. Created by Setup and referenced verbatim by every
// manifest under testdata/.
const TestNamespace = "kgw-e2e-basic-routing"

// Setup is the suite-scoped fixture: a Namespace manifest that creates
// TestNamespace before any scenario runs. Use as Options.VersionedSetup or
// as a base when composing with additional setup.
var Setup = kgwtest.VersionedSetup{
	Default: kgwtest.Setup{
		Manifests: []string{"testdata/_suite.yaml"},
	},
}

// Tests is the registry of scenarios in this package. Populated by init()
// functions in each scenario file. Pass to Suite.Run.
var Tests []kgwtest.Test
