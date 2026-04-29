package kgwtest

import "strings"

// gwApiV1_5_1 is the version at which ListenerSet was promoted from the
// experimental v1alpha1 group to the standard v1 group. Below this version
// on the experimental channel, manifests must use the legacy
// XListenerSet / gateway.networking.x-k8s.io/v1alpha1 form.
const gwApiV1_5_1 = "1.5.1"

// usesLegacyXListenerSet reports whether the suite is running against an
// experimental Gateway API release older than 1.5.1, where ListenerSet was
// still XListenerSet under gateway.networking.x-k8s.io/v1alpha1.
func (s *Suite) usesLegacyXListenerSet() bool {
	if s.apiChannel != ChannelExperimental {
		return false
	}
	if s.apiVersion == "" {
		return false
	}
	cmp, err := compareVersions(s.apiVersion, gwApiV1_5_1)
	if err != nil {
		return false
	}
	return cmp < 0
}

// TransformListenerSetForGwApiVersion rewrites a manifest's ListenerSet
// references to the legacy XListenerSet / gateway.networking.x-k8s.io/v1alpha1
// form when the detected suite is on the experimental channel below 1.5.1.
// On any newer or standard release the input is returned unchanged, so the
// underlying manifest stays kubectl-applyable as written.
//
// The rewrite affects:
//   - "kind: ListenerSet" -> "kind: XListenerSet"
//   - "apiVersion: gateway.networking.k8s.io/v1" preceding "kind: XListenerSet"
//     -> "apiVersion: gateway.networking.x-k8s.io/v1alpha1"
//   - "group: gateway.networking.k8s.io" adjacent to "kind: XListenerSet"
//     -> "group: gateway.networking.x-k8s.io"
func TransformListenerSetForGwApiVersion(s *Suite, in []byte) []byte {
	if !s.usesLegacyXListenerSet() {
		return in
	}

	lines := strings.Split(string(in), "\n")
	for i, line := range lines {
		if strings.TrimSpace(line) == "kind: ListenerSet" {
			lines[i] = strings.Replace(line, "ListenerSet", "XListenerSet", 1)
		}
	}
	for i, line := range lines {
		if strings.TrimSpace(line) == "apiVersion: gateway.networking.k8s.io/v1" &&
			i+1 < len(lines) &&
			strings.TrimSpace(lines[i+1]) == "kind: XListenerSet" {
			lines[i] = strings.Replace(line, "gateway.networking.k8s.io/v1", "gateway.networking.x-k8s.io/v1alpha1", 1)
		}
	}
	for i, line := range lines {
		if strings.TrimSpace(line) != "group: gateway.networking.k8s.io" {
			continue
		}
		hasXListenerSetNeighbor := i > 0 && strings.TrimSpace(lines[i-1]) == "kind: XListenerSet" ||
			i+1 < len(lines) && strings.TrimSpace(lines[i+1]) == "kind: XListenerSet"
		if hasXListenerSetNeighbor {
			lines[i] = strings.Replace(line, "gateway.networking.k8s.io", "gateway.networking.x-k8s.io", 1)
		}
	}

	return []byte(strings.Join(lines, "\n"))
}
