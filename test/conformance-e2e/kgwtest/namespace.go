package kgwtest

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ensureTestNamespace creates Test.Namespace if set and registers cleanup.
// Returns the namespace name, or "" if Test.Namespace is empty (the test
// relies on shared namespaces created by VersionedSetup).
func (s *Suite) ensureTestNamespace(t *testing.T, test *Test) string {
	t.Helper()
	if test.Namespace == "" {
		return ""
	}
	ns := test.Namespace

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

// applyManifests reads each manifest path from Suite.ManifestFS, applies
// the provided transforms (in order) to its bytes, parses the result, and
// applies the resulting objects via the conformance Applier. With no
// transforms the manifest is applied verbatim — kubectl-applyable directly
// from disk.
func (s *Suite) applyManifests(t *testing.T, paths []string, transforms []ManifestTransform) {
	t.Helper()

	for _, path := range paths {
		objects := s.parseManifest(t, path, transforms)
		clientObjs := make([]client.Object, len(objects))
		for i := range objects {
			clientObjs[i] = &objects[i]
		}
		s.Conformance.Applier.MustApplyObjectsWithCleanup(t, s.Conformance.Client, s.Conformance.TimeoutConfig, clientObjs, true)
	}
}

// parseManifest reads a manifest from Suite.ManifestFS, runs the bytes
// through any transforms in order, and decodes the result into a slice of
// unstructured objects.
func (s *Suite) parseManifest(t *testing.T, path string, transforms []ManifestTransform) []unstructured.Unstructured {
	t.Helper()

	raw, err := readFromFS(s.Conformance.ManifestFS, path)
	require.NoErrorf(t, err, "reading manifest %q", path)

	for _, tf := range transforms {
		raw = tf(s, raw)
	}

	decoder := yaml.NewYAMLOrJSONDecoder(bytes.NewReader(raw), 4096)
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
