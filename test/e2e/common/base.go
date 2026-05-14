//go:build e2e

package common

import (
	"context"
	"fmt"
	"net"
	"os"
	"testing"

	"istio.io/istio/pkg/log"
	"istio.io/istio/pkg/test/util/assert"
	"istio.io/istio/pkg/test/util/retry"
	"k8s.io/apimachinery/pkg/types"

	"github.com/kgateway-dev/kgateway/v2/pkg/utils/requestutils/curl"
	"github.com/kgateway-dev/kgateway/v2/test/e2e"
	"github.com/kgateway-dev/kgateway/v2/test/gomega/matchers"
)

func SetupBaseConfig(ctx context.Context, t *testing.T, installation *e2e.TestInstallation, manifests ...string) {
	for _, s := range log.Scopes() {
		s.SetOutputLevel(log.DebugLevel)
	}
	err := installation.ClusterContext.IstioClient.ApplyYAMLFiles("", manifests...)
	assert.NoError(t, err)
}

func SetupBaseGateway(ctx context.Context, t *testing.T, installation *e2e.TestInstallation, name types.NamespacedName) {
	BaseGateway = LookupGateway(ctx, t, installation, name)
}

// LookupGateway resolves a Gateway's external address and returns a Gateway
// value scoped to that name+address. Suites that own their gateway (so they
// can run in parallel against a private gateway without sharing
// common.BaseGateway with other suites) populate a local Gateway field via
// this helper in SetupSuite.
func LookupGateway(ctx context.Context, t *testing.T, installation *e2e.TestInstallation, name types.NamespacedName) Gateway {
	address := installation.AssertionsT(t).EventuallyGatewayAddress(
		ctx,
		name.Name,
		name.Namespace,
	)
	// Allow overriding the gateway address for environments where the LB IP
	// is not directly reachable from the host (e.g., k3d on macOS using port mapping).
	if override := os.Getenv("GATEWAY_ADDRESS_OVERRIDE"); override != "" {
		address = override
	}
	return Gateway{
		NamespacedName: name,
		Address:        address,
	}
}

type Gateway struct {
	types.NamespacedName
	Address string
}

var BaseGateway Gateway

func (g *Gateway) Send(t *testing.T, match *matchers.HttpResponse, opts ...curl.Option) {
	var hostOpt curl.Option
	if _, _, err := net.SplitHostPort(g.Address); err == nil {
		hostOpt = curl.WithHostPort(g.Address)
	} else {
		hostOpt = curl.WithHost(g.Address)
	}
	fullOpts := append([]curl.Option{hostOpt}, opts...)
	retry.UntilSuccessOrFail(t, func() error {
		r, err := curl.ExecuteRequest(fullOpts...)
		if err != nil {
			return err
		}
		defer r.Body.Close()
		mm := matchers.HaveHttpResponse(match)
		success, err := mm.Match(r)
		if err != nil {
			return err
		}
		if !success {
			return fmt.Errorf("match failed: %v", mm.FailureMessage(r))
		}
		return nil
	})
}
