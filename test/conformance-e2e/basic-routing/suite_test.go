//go:build e2e

package basicrouting_test

import (
	"embed"
	"io/fs"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/kgateway-dev/kgateway/v2/test/conformance-e2e/kgwtest"
)

//go:embed testdata/*.yaml
var manifests embed.FS

// tests is populated by init() functions in scenario files.
var tests []kgwtest.Test

func TestBasicRouting(t *testing.T) {
	s := kgwtest.NewSuite(t, kgwtest.Options{
		GatewayClassName: "kgateway",
		ManifestFS:       []fs.FS{manifests},
	})
	require.NoError(t, s.Run(t, tests))
}
