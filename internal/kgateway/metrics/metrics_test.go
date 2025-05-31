package metrics

import (
	"testing"
)

func TestMain(m *testing.M) {
	m.Run()
}

func setupTest() {
	ResetControllerMetrics()
	ResetTranslatorMetrics()
	ResetSnapshotMetrics()
}
