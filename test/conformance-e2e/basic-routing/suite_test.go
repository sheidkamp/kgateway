//go:build e2e

package basicrouting_test

import (
	"io/fs"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/kgateway-dev/kgateway/v2/test/conformance-e2e/basic-routing"
	"github.com/kgateway-dev/kgateway/v2/test/conformance-e2e/kgwtest"
)

func TestBasicRouting(t *testing.T) {
	s := kgwtest.NewSuite(t, kgwtest.Options{
		GatewayClassName: "kgateway",
		ManifestFS:       []fs.FS{basicrouting.ManifestFS},
		VersionedSetup:   basicrouting.Setup,
	})
	require.NoError(t, s.Run(t, basicrouting.Tests))
}
