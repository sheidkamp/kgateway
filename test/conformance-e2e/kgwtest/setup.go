package kgwtest

import (
	"sort"
	"testing"
)

// Setup is a single suite-scoped fixture configuration: a list of manifest
// paths (resolved against Suite.ManifestFS) to apply once before any test
// runs, plus an optional hook invoked after the manifests are applied.
type Setup struct {
	// Manifests is a list of manifest paths applied via the conformance
	// Applier. Paths are resolved against Suite.ManifestFS.
	Manifests []string

	// PostApply, if set, runs after all Manifests have been applied. Use for
	// waits that can't be expressed declaratively in a manifest (e.g.,
	// waiting for a Deployment created by the Gateway controller).
	PostApply func(t *testing.T, s *Suite)
}

// VersionedSetup picks a Setup based on the detected Gateway API channel and
// version. Selection semantics mirror the existing test/e2e base suite:
// within the matching channel, choose the highest entry with version <=
// detected version; otherwise fall through to Default.
type VersionedSetup struct {
	// Default is applied when no ByVersion entry matches.
	Default Setup

	// ByVersion maps channel -> semver -> Setup. Semver keys omit the "v"
	// prefix (e.g., "1.5.0").
	ByVersion map[Channel]map[string]Setup
}

// selectSetup picks the Setup to apply given the detected channel and version.
// Returns Default if no ByVersion entry applies or parsing fails.
func (vs VersionedSetup) selectSetup(detectedVersion string, detectedChannel Channel) Setup {
	byVersion, ok := vs.ByVersion[detectedChannel]
	if !ok || len(byVersion) == 0 {
		return vs.Default
	}

	versions := make([]string, 0, len(byVersion))
	for v := range byVersion {
		versions = append(versions, v)
	}
	sort.Slice(versions, func(i, j int) bool {
		cmp, err := compareVersions(versions[i], versions[j])
		if err != nil {
			return versions[i] < versions[j]
		}
		return cmp < 0
	})

	var chosen string
	for _, v := range versions {
		cmp, err := compareVersions(v, detectedVersion)
		if err != nil {
			continue
		}
		if cmp <= 0 {
			chosen = v
		}
	}

	if chosen == "" {
		return vs.Default
	}
	return byVersion[chosen]
}
