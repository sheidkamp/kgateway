package kgwtest

import (
	"context"
	"fmt"
	"strings"

	"github.com/Masterminds/semver/v3"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	crdBundleVersionAnnotation = "gateway.networking.k8s.io/bundle-version"
	crdChannelAnnotation       = "gateway.networking.k8s.io/channel"
)

// detectGwApiVersionAndChannel reads the Gateway API CRDs from the cluster and
// returns the installed bundle version and channel. Mirrors upstream's
// getAPIVersionAndChannel but returns an error instead of aborting on mismatch
// — callers decide whether to tolerate mismatches.
func detectGwApiVersionAndChannel(ctx context.Context, c client.Client) (string, Channel, error) {
	var crds apiextensionsv1.CustomResourceDefinitionList
	if err := c.List(ctx, &crds); err != nil {
		return "", "", fmt.Errorf("listing CRDs: %w", err)
	}

	var version, channel string
	for _, crd := range crds.Items {
		if !strings.HasSuffix(crd.Spec.Group, "gateway.networking.k8s.io") {
			continue
		}
		v, okv := crd.Annotations[crdBundleVersionAnnotation]
		ch, okc := crd.Annotations[crdChannelAnnotation]
		if !okv && !okc {
			continue
		}
		if !okv || !okc {
			return "", "", fmt.Errorf("CRD %s has partial version/channel annotations", crd.Name)
		}
		if version != "" && v != version {
			return "", "", fmt.Errorf("multiple Gateway API versions detected: %s vs %s", version, v)
		}
		if channel != "" && ch != channel {
			return "", "", fmt.Errorf("multiple Gateway API channels detected: %s vs %s", channel, ch)
		}
		version, channel = v, ch
	}

	if version == "" || channel == "" {
		return "", "", fmt.Errorf("no Gateway API CRDs with version/channel annotations found")
	}

	return version, Channel(channel), nil
}

// compareVersions returns -1, 0, or 1 if a is less than, equal to, or greater
// than b. Accepts semver strings with or without a leading "v".
func compareVersions(a, b string) (int, error) {
	av, err := semver.NewVersion(a)
	if err != nil {
		return 0, fmt.Errorf("parsing %q: %w", a, err)
	}
	bv, err := semver.NewVersion(b)
	if err != nil {
		return 0, fmt.Errorf("parsing %q: %w", b, err)
	}
	return av.Compare(bv), nil
}

// checkVersionBounds returns a skip reason if the detected version/channel
// fails to satisfy the given bounds. An empty reason means the test is eligible
// to run.
func checkVersionBounds(detectedVersion string, detectedChannel Channel, minV, maxV string, requireChannel Channel) string {
	if requireChannel != "" && detectedChannel != requireChannel {
		return fmt.Sprintf("requires channel %q, detected %q", requireChannel, detectedChannel)
	}
	if minV != "" {
		cmp, err := compareVersions(detectedVersion, minV)
		if err != nil {
			return fmt.Sprintf("cannot compare detected version %q with min %q: %v", detectedVersion, minV, err)
		}
		if cmp < 0 {
			return fmt.Sprintf("requires Gateway API >= %s, detected %s", minV, detectedVersion)
		}
	}
	if maxV != "" {
		cmp, err := compareVersions(detectedVersion, maxV)
		if err != nil {
			return fmt.Sprintf("cannot compare detected version %q with max %q: %v", detectedVersion, maxV, err)
		}
		if cmp > 0 {
			return fmt.Sprintf("requires Gateway API <= %s, detected %s", maxV, detectedVersion)
		}
	}
	return ""
}
