package kgwtest

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"strings"
	"testing"
	"text/template"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// namespacePrefix is prepended to ShortName to form the default per-test
// namespace name. Kept short so we don't bump into the 63-char k8s name limit
// for reasonable ShortNames.
const namespacePrefix = "kgw-e2e-"

// namespaceForTest returns the namespace name this test should use.
func namespaceForTest(t *Test) string {
	if t.Namespace != "" {
		return t.Namespace
	}
	return namespacePrefix + slug(t.ShortName)
}

// slug normalizes a ShortName into a DNS-label-safe string.
func slug(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '-':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r + ('a' - 'A'))
		default:
			b.WriteRune('-')
		}
	}
	out := strings.Trim(b.String(), "-")
	if len(out) > 63 {
		out = out[:63]
	}
	return out
}

// ensureTestNamespace creates the per-test namespace and registers cleanup.
func (s *Suite) ensureTestNamespace(t *testing.T, test *Test) string {
	t.Helper()
	ns := namespaceForTest(test)

	obj := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: ns},
	}

	ctx, cancel := context.WithTimeout(context.Background(), s.Conformance.TimeoutConfig.CreateTimeout)
	defer cancel()

	err := s.Conformance.Client.Create(ctx, obj)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		require.NoError(t, err, "creating per-test namespace %q", ns)
	}

	t.Cleanup(func() {
		delCtx, delCancel := context.WithTimeout(context.Background(), s.Conformance.TimeoutConfig.DeleteTimeout)
		defer delCancel()
		if err := s.Conformance.Client.Delete(delCtx, obj); err != nil && !apierrors.IsNotFound(err) {
			t.Logf("kgwtest: error deleting per-test namespace %q: %v", ns, err)
		}
	})

	return ns
}

// applyManifestsInNamespace reads each manifest path from Suite.ManifestFS,
// renders it through text/template with TestNamespace and GatewayClassName,
// and applies the resulting objects via the conformance Applier.
func (s *Suite) applyManifestsInNamespace(t *testing.T, test *Test, ns string) {
	t.Helper()

	for _, path := range test.Manifests {
		objects := s.renderManifest(t, path, ns)
		clientObjs := make([]client.Object, len(objects))
		for i := range objects {
			clientObjs[i] = &objects[i]
		}
		s.Conformance.Applier.MustApplyObjectsWithCleanup(t, s.Conformance.Client, s.Conformance.TimeoutConfig, clientObjs, true)
	}
}

// renderManifest reads a manifest from Suite.ManifestFS, applies templating,
// and parses the result into unstructured objects. Also overrides
// Gateway.spec.gatewayClassName so manifests don't need to hard-code it.
func (s *Suite) renderManifest(t *testing.T, path, ns string) []unstructured.Unstructured {
	t.Helper()

	raw, err := readFromFS(s.Conformance.ManifestFS, path)
	require.NoErrorf(t, err, "reading manifest %q", path)

	tpl, err := template.New(path).Option("missingkey=error").Parse(string(raw))
	require.NoErrorf(t, err, "parsing template %q", path)

	data := map[string]string{
		"TestNamespace":    ns,
		"GatewayClassName": s.opts.GatewayClassName,
	}

	var buf bytes.Buffer
	require.NoErrorf(t, tpl.Execute(&buf, data), "executing template %q", path)

	decoder := yaml.NewYAMLOrJSONDecoder(&buf, 4096)
	var objects []unstructured.Unstructured
	for {
		obj := unstructured.Unstructured{}
		if err := decoder.Decode(&obj); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			require.NoErrorf(t, err, "decoding manifest %q", path)
		}
		if len(obj.Object) == 0 {
			continue
		}
		if obj.GetKind() == "Gateway" && obj.GetAPIVersion() == "gateway.networking.k8s.io/v1" {
			require.NoError(t, unstructured.SetNestedField(obj.Object, s.opts.GatewayClassName, "spec", "gatewayClassName"))
		}
		objects = append(objects, obj)
	}
	return objects
}

// readFromFS reads a path from the first fs.FS that contains it. Mirrors
// upstream's behavior of treating ManifestFS as an ordered search path.
func readFromFS(manifestFS []fs.FS, path string) ([]byte, error) {
	if len(manifestFS) == 0 {
		return nil, fmt.Errorf("no manifest filesystem configured")
	}
	var lastErr error
	for _, mfs := range manifestFS {
		data, err := fs.ReadFile(mfs, path)
		if err == nil {
			return data, nil
		}
		lastErr = err
	}
	return nil, fmt.Errorf("manifest %q not found: %w", path, lastErr)
}
