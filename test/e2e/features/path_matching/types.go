//go:build e2e

package path_matching

import (
	"path/filepath"

	"github.com/kgateway-dev/kgateway/v2/pkg/utils/fsutils"
	"github.com/kgateway-dev/kgateway/v2/test/e2e/tests/base"
)

var (
	// manifests
	exactManifest         = filepath.Join(fsutils.MustGetThisDir(), "testdata", "exact.yaml")
	prefixManifest        = filepath.Join(fsutils.MustGetThisDir(), "testdata", "prefix.yaml")
	regexManifest         = filepath.Join(fsutils.MustGetThisDir(), "testdata", "regex.yaml")
	prefixRewriteManifest = filepath.Join(fsutils.MustGetThisDir(), "testdata", "prefix-rewrite.yaml")

	// The suite needs no setup of its own: it routes through the base gateway to the shared
	// httpbin backend applied at base level by common.SetupSharedHttpbinBackend.
	setup = base.TestCase{}

	// test cases
	testCases = map[string]*base.TestCase{
		"TestExactMatch": {
			Manifests: []string{exactManifest},
		},
		"TestPrefixMatch": {
			Manifests: []string{prefixManifest},
		},
		"TestRegexMatch": {
			Manifests: []string{regexManifest},
		},
		"TestPrefixRewrite": {
			Manifests: []string{prefixRewriteManifest},
		},
	}
)
