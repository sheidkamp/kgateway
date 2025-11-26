package common

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/kgateway-dev/kgateway/v2/pkg/utils/requestutils/curl"
	"github.com/kgateway-dev/kgateway/v2/test/e2e"
	"github.com/kgateway-dev/kgateway/v2/test/gomega/matchers"
	testmatchers "github.com/kgateway-dev/kgateway/v2/test/gomega/matchers"
	"istio.io/istio/pkg/log"
	"istio.io/istio/pkg/test/util/assert"
	"istio.io/istio/pkg/test/util/retry"
	"k8s.io/apimachinery/pkg/types"
)

func SetupBaseConfig(ctx context.Context, t *testing.T, installation *e2e.TestInstallation, manifests ...string) {
	for _, s := range log.Scopes() {
		s.SetOutputLevel(log.DebugLevel)
	}
		err := installation.ClusterContext.IstioClient.ApplyYAMLFiles("", manifests...)
		assert.NoError(t, err)
	//for _, manifest := range manifests {
		//err := installation.Actions.Kubectl().ApplyFile(ctx, manifest)
	//}
}

func SetupBaseGateway(ctx context.Context, installation *e2e.TestInstallation, name types.NamespacedName) {
	address := installation.Assertions.EventuallyGatewayAddress(
		ctx,
		name.Name,
		name.Namespace,
	)
	BaseGateway = Gateway{
		NamespacedName: name,
		Address:        address,
	}
}

type Gateway struct {
	types.NamespacedName
	Address string
}

var BaseGateway Gateway

func (g *Gateway) Send(t *testing.T, match *testmatchers.HttpResponse, opts ...curl.Option) {
	fullOpts := append([]curl.Option{curl.WithHost(g.Address)}, opts...)
	var resp *http.Response
	retry.UntilSuccessOrFail(t, func() error {
		t0 := time.Now()
		log.Errorf("howardjohn: start req")
		r, err := curl.ExecuteRequest(fullOpts...)
		if err != nil {
			return err
		}
		log.Errorf("howardjohn: start end req %v", time.Since(t0))
		resp = r
		mm := matchers.HaveHttpResponse(match)
		success, err := mm.Match(resp)
		if err != nil {
			return err
		}
		if !success {
			return fmt.Errorf("match failed: %v", mm.FailureMessage(resp))
		}
		log.Errorf("howardjohn: success")
		return nil
	})
}