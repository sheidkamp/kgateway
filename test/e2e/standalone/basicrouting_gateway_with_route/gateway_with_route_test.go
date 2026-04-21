//go:build e2e

package basicroutinggatewaywithroute_test

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlcfg "sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/e2e-framework/klient"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/kgateway-dev/kgateway/v2/pkg/utils/requestutils/curl"
	testdefaults "github.com/kgateway-dev/kgateway/v2/test/e2e/defaults"
)

const (
	gatewayName      = "gateway"
	gatewayNamespace = "default"
	hostHeader       = "example.com"

	listenerHTTPAlt = 8080
	listenerHTTP    = 80
)

func TestGatewayWithRoute(t *testing.T) {
	t.Helper()

	ctx := context.Background()

	restConfig, err := ctrlcfg.GetConfig()
	require.NoError(t, err)

	k8sClient, err := klient.New(restConfig)
	require.NoError(t, err)

	scheme := runtime.NewScheme()
	require.NoError(t, gwv1.Install(scheme))

	apiReader, err := ctrlclient.New(restConfig, ctrlclient.Options{Scheme: scheme})
	require.NoError(t, err)

	cfg := envconf.New().WithClient(k8sClient)

	testEnv, err := env.NewWithContext(ctx, cfg)
	require.NoError(t, err)

	var gatewayAddress string

	feature := features.New("GatewayWithRoute").
		WithLabel("framework", "sigs-e2e-framework").
		WithLabel("scenario", "basicrouting-gateway-with-route").
		Setup(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			t.Helper()
			gatewayAddress = waitForGatewayAddress(ctx, t, cfg, apiReader)
			return ctx
		}).
		Assess("HTTP 200 on listener 8080", func(ctx context.Context, t *testing.T, _ *envconf.Config) context.Context {
			_ = ctx
			assertHTTP200(t, gatewayAddress, listenerHTTPAlt)
			return ctx
		}).
		Assess("HTTP 200 on listener 80", func(ctx context.Context, t *testing.T, _ *envconf.Config) context.Context {
			_ = ctx
			assertHTTP200(t, gatewayAddress, listenerHTTP)
			return ctx
		}).
		Feature()

	testEnv.Test(t, feature)
}

func waitForGatewayAddress(ctx context.Context, t *testing.T, _ *envconf.Config, apiReader ctrlclient.Client) string {
	t.Helper()

	var gatewayAddress string

	err := wait.For(func(ctx context.Context) (bool, error) {
		var gateway gwv1.Gateway
		if getErr := apiReader.Get(ctx, ctrlclient.ObjectKey{Name: gatewayName, Namespace: gatewayNamespace}, &gateway); getErr != nil {
			return false, getErr
		}

		address, ok := extractGatewayAddress(&gateway)
		if !ok {
			return false, nil
		}

		gatewayAddress = address
		return true, nil
	}, wait.WithContext(ctx), wait.WithTimeout(2*time.Minute), wait.WithInterval(2*time.Second))
	require.NoError(t, err)

	return gatewayAddress
}

func extractGatewayAddress(gateway *gwv1.Gateway) (string, bool) {
	for _, address := range gateway.Status.Addresses {
		if address.Value == "" {
			continue
		}
		return address.Value, true
	}

	return "", false
}

func assertHTTP200(t *testing.T, gatewayAddress string, port int) {
	t.Helper()

	response, err := curl.ExecuteRequest(
		curl.WithHost(gatewayAddress),
		curl.WithHostHeader(hostHeader),
		curl.WithPort(port),
	)
	require.NoError(t, err)
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	require.NoError(t, err)

	require.Equal(t, http.StatusOK, response.StatusCode, "unexpected status for %s:%d", gatewayAddress, port)
	require.Contains(t, strings.TrimSpace(string(body)), testdefaults.NginxResponse, "unexpected body for %s:%d", gatewayAddress, port)
}
