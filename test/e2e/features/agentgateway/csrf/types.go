//go:build e2e

package csrf

import (
	"path/filepath"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kgatewayv1alpha1 "github.com/kgateway-dev/kgateway/v2/api/v1alpha1"
	"github.com/kgateway-dev/kgateway/v2/pkg/utils/fsutils"
)

const (
	// test namespace for proxy resources
	namespace = "agentgateway-base"
)

var (
	// manifests
	routesManifest      = getTestFile("routes.yaml")
	csrfAgwPolicyManifest = getTestFile("csrf-gw.yaml")

	agwPolicy = &kgatewayv1alpha1.AgentgatewayPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "csrf-gw-policy",
			Namespace: namespace,
		},
	}
)

func getTestFile(filename string) string {
	return filepath.Join(fsutils.MustGetThisDir(), "testdata", filename)
}
