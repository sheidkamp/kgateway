package metrics_test

import (
	"testing"

	. "github.com/kgateway-dev/kgateway/v2/internal/kgateway/metrics"
)

func TestMain(m *testing.M) {
	m.Run()
}

func setupTest() {
	ResetCollectionMetrics()
	ResetControllerMetrics()
	ResetTranslatorMetrics()
	ResetStatusSyncMetrics()
	ResetRoutingMetrics()
}
