//go:build e2e

package common

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"testing"
	"time"

	"istio.io/istio/pkg/log"
	"istio.io/istio/pkg/test/util/retry"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	utilyaml "k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kgateway-dev/kgateway/v2/test/e2e"
	testdefaults "github.com/kgateway-dev/kgateway/v2/test/e2e/defaults"
	"github.com/kgateway-dev/kgateway/v2/test/testutils"
)

// SharedNginxNamespace is the namespace of the shared nginx backend applied by
// SetupSharedNginxBackend.
const SharedNginxNamespace = "nginx-shared"

func SetupBaseConfig(ctx context.Context, t *testing.T, installation *e2e.TestInstallation, manifests ...string) {
	for _, s := range log.Scopes() {
		s.SetOutputLevel(log.DebugLevel)
	}
	// Register cleanup before applying so partially applied manifests are still removed.
	testutils.Cleanup(t, func() {
		// Delete through the Istio client so resources whose manifests omit metadata.namespace
		// resolve to the same default namespace the apply used.
		if err := installation.ClusterContext.IstioClient.DeleteYAMLFiles("", manifests...); err != nil {
			t.Logf("failed to delete base config manifests %v: %v", manifests, err)
		}
		// Wait for the namespaces these manifests create to be fully deleted before returning, so a
		// gotestsum rerun (a fresh process) does not try to apply into a still-terminating namespace.
		waitCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		if err := waitForManifestNamespacesDeleted(waitCtx, installation.ClusterContext.Client, manifests...); err != nil {
			t.Logf("failed waiting for base config namespaces to delete: %v", err)
		}
	})
	// Apply manifests one at a time to avoid the concurrent-apply race where a namespace and
	// namespace-scoped resources are applied simultaneously and the scoped resources are created
	// before the namespace exists.
	for _, manifest := range manifests {
		if err := installation.ClusterContext.IstioClient.ApplyYAMLFiles("", manifest); err != nil {
			t.Fatalf("apply manifest %s: %v", manifest, err)
		}
	}
}

// namespacesFromManifests parses the given manifest files and returns the names of every Namespace
// they declare. Documents that are not a named Namespace are ignored, so empty or malformed YAML
// documents are harmless.
func namespacesFromManifests(manifests ...string) ([]string, error) {
	var names []string
	for _, manifest := range manifests {
		data, err := os.ReadFile(manifest)
		if err != nil {
			return nil, fmt.Errorf("read manifest %s: %w", manifest, err)
		}
		decoder := utilyaml.NewYAMLOrJSONDecoder(bytes.NewReader(data), 4096)
		for {
			obj := &unstructured.Unstructured{}
			if err := decoder.Decode(obj); err != nil {
				if err == io.EOF {
					break
				}
				return nil, fmt.Errorf("decode manifest %s: %w", manifest, err)
			}
			if obj.GetObjectKind().GroupVersionKind().Kind == "Namespace" && obj.GetName() != "" {
				names = append(names, obj.GetName())
			}
		}
	}
	return names, nil
}

// waitForManifestNamespacesDeleted polls until every Namespace declared in the given manifests is
// absent from the cluster, or ctx expires. Only Namespaces are checked: they are the slow,
// race-prone part of teardown, and being cluster-scoped they can be looked up by name alone (no
// namespace needed), avoiding the empty-namespace problem that affects namespace-scoped resources
// whose manifests omit metadata.namespace.
func waitForManifestNamespacesDeleted(ctx context.Context, c client.Client, manifests ...string) error {
	names, err := namespacesFromManifests(manifests...)
	if err != nil {
		return err
	}
	if len(names) == 0 {
		return nil
	}
	// Honor the caller's deadline for the retry loop; fall back to 2 minutes if ctx has none.
	timeout := 2 * time.Minute
	if deadline, ok := ctx.Deadline(); ok {
		timeout = time.Until(deadline)
	}
	return retry.UntilSuccess(func() error {
		for _, name := range names {
			ns := &corev1.Namespace{}
			err := c.Get(ctx, client.ObjectKey{Name: name}, ns)
			if err == nil {
				return fmt.Errorf("namespace %s still exists", name)
			}
			if !apierrors.IsNotFound(err) {
				return fmt.Errorf("get namespace %s: %w", name, err)
			}
		}
		return nil
	}, retry.Timeout(timeout), retry.Delay(2*time.Second))
}

// SetupSharedNginxBackend applies the shared nginx pod (ns nginx-shared)
func SetupSharedNginxBackend(ctx context.Context, t *testing.T, installation *e2e.TestInstallation) {
	SetupBaseConfig(ctx, t, installation, testdefaults.NginxPodManifest)
	installation.AssertionsT(t).EventuallyPodsRunning(ctx, SharedNginxNamespace, metav1.ListOptions{
		LabelSelector: testdefaults.WellKnownAppLabel + "=nginx",
	}, 2*time.Minute)
}

// SetupBaseGateway resolves the LB address for the named Gateway and stores it in BaseGateway.
//
// GATEWAY_ADDRESS_OVERRIDE: when set, overrides the resolved address. This exists to support
// environments where the LB IP is not directly reachable from the host (e.g., k3d on macOS using
// port mapping). The override is applied ONLY here — single-gateway suites that use BaseGateway
// pick it up automatically. Suites that construct their own common.Gateway values (e.g.,
// multi-gateway suites that need more than one address) do NOT honor the override, since a single
// env var cannot disambiguate multiple gateways. Running such suites under k3d is out of scope.
func SetupBaseGateway(ctx context.Context, t *testing.T, installation *e2e.TestInstallation, name types.NamespacedName) {
	address := installation.AssertionsT(t).EventuallyGatewayAddress(
		ctx,
		name.Name,
		name.Namespace,
	)
	if override := os.Getenv("GATEWAY_ADDRESS_OVERRIDE"); override != "" {
		address = override
	}
	BaseGateway = Gateway{
		NamespacedName: name,
		Address:        address,
	}
}

var BaseGateway Gateway
